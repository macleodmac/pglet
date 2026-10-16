package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	DefaultModel = "claude-haiku-5-5"

	maxTokens = 16000

	// selectTablesThreshold is the number of tables above which we pre-filter
	// to relevant tables before generating SQL.
	selectTablesThreshold = 20

	maxEnumValues = 30
	maxCommentLen = 120
	ellipsis      = "…"
)

var ErrRefused = errors.New("the model declined this request")

// Table describes a table, view or materialized view for schema context.
type Table struct {
	Name    string // schema-qualified
	Kind    string
	Comment string
	Columns []Column
}

// Column describes a column for schema context.
type Column struct {
	Name       string
	Type       string
	PrimaryKey bool
	References string // schema.table.column of the foreign key target
	EnumValues []string
	Comment    string
}

// Message represents a conversation turn.
type Message struct {
	Role    string
	Content string
}

// Client generates SQL with Claude.
type Client struct {
	api   anthropic.Client
	Model string
}

// NewClient creates a Client with sensible defaults.
func NewClient(apiKey string) *Client {
	return &Client{
		api:   anthropic.NewClient(option.WithAPIKey(apiKey)),
		Model: DefaultModel,
	}
}

type sqlResult struct {
	SQL         string `json:"sql"`
	Explanation string `json:"explanation"`
}

var sqlResultSchema = objectSchema(map[string]any{
	"sql":         map[string]any{"type": "string"},
	"explanation": map[string]any{"type": "string"},
})

// GenerateSQL turns a natural-language prompt into a PostgreSQL query.
// For large schemas it first selects relevant tables using a cheap call.
func (c *Client) GenerateSQL(ctx context.Context, schema []Table, prompt string, history []Message) (sql, explanation string, err error) {
	if len(schema) > selectTablesThreshold {
		var asked []string
		for _, m := range history {
			if m.Role == "user" {
				asked = append(asked, m.Content)
			}
		}
		schema = c.selectTables(ctx, schema, strings.Join(append(asked, prompt), "\n"))
	}

	system := fmt.Sprintf(`You are a PostgreSQL SQL query generator. Given a natural language description, generate a valid PostgreSQL query.

Database schema (PK = primary key, -> = foreign key target):
%s
Rules:
- Generate only valid PostgreSQL SQL
- Use the exact table and column names from the schema
- Join along the foreign keys shown, and filter enum columns using only their listed values
- If the request is ambiguous, make reasonable assumptions and explain them
- Keep the explanation brief`, formatSchema(schema))

	res, err := completeJSON[sqlResult](ctx, c, system, buildMessages(history, prompt), anthropic.OutputConfigEffortMedium, sqlResultSchema)
	if err != nil {
		return "", "", err
	}
	return res.SQL, res.Explanation, nil
}

// FixSQL corrects a query given the error PostgreSQL returned for it.
func (c *Client) FixSQL(ctx context.Context, schema []Table, query, errMsg string) (sql, explanation string, err error) {
	if len(schema) > selectTablesThreshold {
		schema = c.selectTables(ctx, schema, query+"\n"+errMsg)
	}

	system := fmt.Sprintf(`You fix PostgreSQL queries that failed. Given a query and the error PostgreSQL returned for it, return a corrected query.

Database schema (PK = primary key, -> = foreign key target):
%s
Rules:
- Fix the reported error and anything else in the schema that would stop the query from running
- Otherwise keep the query's intent, formatting and comments
- Use the exact table and column names from the schema
- In the explanation, say briefly what was wrong and what you changed`, formatSchema(schema))

	msgs := []Message{{Role: "user", Content: fmt.Sprintf("Query:\n%s\n\nError:\n%s", query, errMsg)}}
	res, err := completeJSON[sqlResult](ctx, c, system, msgs, anthropic.OutputConfigEffortMedium, sqlResultSchema)
	if err != nil {
		return "", "", err
	}
	return res.SQL, res.Explanation, nil
}

// GenerateTabName produces a short (2-4 word) name for a SQL query tab.
func (c *Client) GenerateTabName(ctx context.Context, sql string) (string, error) {
	system := `You generate very short tab names for SQL queries in a database browser.
Rules:
- Return ONLY the tab name, nothing else
- 2-4 words maximum
- Describe what the query does, not the SQL syntax
- Use title case
- Examples: "Active Users", "Monthly Revenue", "Order Details", "Schema Sizes"`

	name, err := c.complete(ctx, system, []Message{{Role: "user", Content: sql}}, anthropic.OutputConfigEffortLow, nil)
	if err != nil {
		return "", err
	}
	return strings.Trim(strings.TrimSpace(name), "\"'`"), nil
}

// GenerateSuggestions produces 4 contextual query suggestions based on table names.
func (c *Client) GenerateSuggestions(ctx context.Context, tables []string) ([]string, error) {
	system := `You generate short, natural-language query suggestions for a PostgreSQL database browser.
Rules:
- Return exactly 4 suggestions
- Each suggestion should be a natural-language prompt a user might type to generate SQL
- Use actual table names from the provided list
- Keep each suggestion under 50 characters
- Cover different query types: aggregation, filtering, joins, listing
- Do not include SQL syntax, just natural language`

	msgs := []Message{{Role: "user", Content: "Tables: " + strings.Join(tables, ", ")}}
	res, err := completeJSON[struct {
		Suggestions []string `json:"suggestions"`
	}](ctx, c, system, msgs, anthropic.OutputConfigEffortLow, objectSchema(map[string]any{
		"suggestions": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	}))
	if err != nil {
		return nil, err
	}
	return res.Suggestions, nil
}

// selectTables asks for the tables relevant to the request.
// Falls back to the full schema on any error.
func (c *Client) selectTables(ctx context.Context, schema []Table, request string) []Table {
	names := make([]string, len(schema))
	for i, t := range schema {
		names[i] = t.Name
	}

	system := `You select which database tables are relevant to a user's request.
Return the schema-qualified table names (e.g. "public.users"). Include tables that might be needed for JOINs. When in doubt, include the table.`

	msgs := []Message{{Role: "user", Content: fmt.Sprintf("Tables: %s\n\nRequest: %s", strings.Join(names, ", "), request)}}
	res, err := completeJSON[struct {
		Tables []string `json:"tables"`
	}](ctx, c, system, msgs, anthropic.OutputConfigEffortLow, objectSchema(map[string]any{
		"tables": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	}))
	if err != nil || len(res.Tables) == 0 {
		return schema
	}

	selected := make(map[string]bool, len(res.Tables))
	for _, name := range res.Tables {
		selected[name] = true
	}
	var filtered []Table
	for _, t := range schema {
		if selected[t.Name] {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) == 0 {
		return schema
	}
	return filtered
}

func completeJSON[T any](ctx context.Context, c *Client, system string, msgs []Message, effort anthropic.OutputConfigEffort, schema map[string]any) (T, error) {
	var out T
	text, err := c.complete(ctx, system, msgs, effort, schema)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return out, fmt.Errorf("parse response: %w", err)
	}
	return out, nil
}

// complete sends a request to the Messages API and returns the response text.
// A non-nil schema constrains the response to JSON matching it.
func (c *Client) complete(ctx context.Context, system string, msgs []Message, effort anthropic.OutputConfigEffort, schema map[string]any) (string, error) {
	params := anthropic.MessageNewParams{
		Model:        c.Model,
		MaxTokens:    maxTokens,
		System:       []anthropic.TextBlockParam{{Text: system}},
		Messages:     make([]anthropic.MessageParam, len(msgs)),
		OutputConfig: anthropic.OutputConfigParam{Effort: effort},
	}
	for i, m := range msgs {
		block := anthropic.NewTextBlock(m.Content)
		if m.Role == "assistant" {
			params.Messages[i] = anthropic.NewAssistantMessage(block)
		} else {
			params.Messages[i] = anthropic.NewUserMessage(block)
		}
	}
	if schema != nil {
		params.OutputConfig.Format = anthropic.JSONOutputFormatParam{Schema: schema}
	}

	resp, err := c.api.Messages.New(ctx, params)
	if err != nil {
		return "", err
	}
	switch resp.StopReason {
	case anthropic.StopReasonRefusal:
		return "", ErrRefused
	case anthropic.StopReasonMaxTokens:
		return "", errors.New("response was cut off before it finished")
	}

	// Thinking blocks can precede the answer, so read text blocks by type.
	var b strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			b.WriteString(t.Text)
		}
	}
	if b.Len() == 0 {
		return "", errors.New("empty response from API")
	}
	return b.String(), nil
}

func objectSchema(properties map[string]any) map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             slices.Sorted(maps.Keys(properties)),
		"additionalProperties": false,
	}
}

func buildMessages(history []Message, prompt string) []Message {
	msgs := make([]Message, 0, len(history)+1)
	msgs = append(msgs, history...)
	return append(msgs, Message{Role: "user", Content: prompt})
}

// formatSchema renders tables one column per line, e.g.
//
//	public.orders (table) -- customer orders
//	  id int8 PK
//	  user_id int8 -> public.users.id
//	  status order_status enum('pending', 'paid')
func formatSchema(schema []Table) string {
	var b strings.Builder
	for _, t := range schema {
		fmt.Fprintf(&b, "%s (%s)", t.Name, t.Kind)
		writeComment(&b, t.Comment)
		b.WriteByte('\n')
		for _, col := range t.Columns {
			fmt.Fprintf(&b, "  %s %s", col.Name, shortType(col.Type))
			if col.PrimaryKey {
				b.WriteString(" PK")
			}
			if col.References != "" {
				b.WriteString(" -> " + col.References)
			}
			if len(col.EnumValues) > 0 {
				writeEnum(&b, col.EnumValues)
			}
			writeComment(&b, col.Comment)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func writeEnum(b *strings.Builder, values []string) {
	b.WriteString(" enum(")
	for i, v := range values {
		if i == maxEnumValues {
			b.WriteString(", " + ellipsis)
			break
		}
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("'" + strings.ReplaceAll(v, "'", "''") + "'")
	}
	b.WriteByte(')')
}

func writeComment(b *strings.Builder, comment string) {
	comment = strings.Join(strings.Fields(comment), " ")
	if comment == "" {
		return
	}
	if r := []rune(comment); len(r) > maxCommentLen {
		comment = string(r[:maxCommentLen]) + ellipsis
	}
	b.WriteString(" -- " + comment)
}

func shortType(t string) string {
	switch strings.ToLower(t) {
	case "integer":
		return "int"
	case "bigint":
		return "int8"
	case "smallint":
		return "int2"
	case "boolean":
		return "bool"
	case "character varying":
		return "varchar"
	case "character":
		return "char"
	case "timestamp without time zone":
		return "timestamp"
	case "timestamp with time zone":
		return "timestamptz"
	case "double precision":
		return "float8"
	case "real":
		return "float4"
	default:
		return t
	}
}
