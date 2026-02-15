package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	DefaultModel     = "claude-haiku-4-5-20251001"
	DefaultFastModel = "claude-haiku-4-5-20251001"

	apiURL     = "https://api.anthropic.com/v1/messages"
	apiVersion = "2023-06-01"

	// selectTablesThreshold is the number of tables above which we pre-filter
	// to relevant tables before generating SQL.
	selectTablesThreshold = 20
)

// Column is a minimal column descriptor used for schema context.
type Column struct {
	Name string
	Type string
}

// Message represents a conversation turn.
type Message struct {
	Role    string
	Content string
}

// Client is an Anthropic API client for SQL generation.
type Client struct {
	APIKey     string
	Model      string
	FastModel  string
	HTTPClient *http.Client
}

// NewClient creates a Client with sensible defaults.
func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:     apiKey,
		Model:      DefaultModel,
		FastModel:  DefaultFastModel,
		HTTPClient: http.DefaultClient,
	}
}

// GenerateSQL turns a natural-language prompt into a PostgreSQL query.
// For large schemas (>20 tables) it first selects relevant tables using a fast model.
func (c *Client) GenerateSQL(ctx context.Context, schema map[string][]Column, prompt string, history []Message) (sql, explanation string, err error) {
	if len(schema) > selectTablesThreshold {
		schema = c.selectTables(ctx, schema, prompt)
	}

	system := fmt.Sprintf(`You are a PostgreSQL SQL query generator. Given a natural language description, generate a valid PostgreSQL query.

Database schema:
%s
Rules:
- Generate only valid PostgreSQL SQL
- Use the exact table and column names from the schema
- Return a JSON object with "sql" (the query) and "explanation" (brief explanation)
- Do not include markdown code fences in the SQL
- If the request is ambiguous, make reasonable assumptions and explain them`, formatSchema(schema))

	msgs := buildMessages(history, prompt)

	text, err := c.complete(ctx, c.Model, system, msgs, 2048)
	if err != nil {
		return "", "", err
	}

	var resp struct {
		SQL         string `json:"sql"`
		Explanation string `json:"explanation"`
	}
	if err := extractJSON(text, &resp); err == nil && resp.SQL != "" {
		return resp.SQL, resp.Explanation, nil
	}

	// Fallback: treat the whole response as SQL.
	return text, "Generated from natural language prompt", nil
}

// complete sends a request to the Anthropic messages API.
func (c *Client) complete(ctx context.Context, model, system string, messages []Message, maxTokens int) (string, error) {
	apiMsgs := make([]map[string]string, len(messages))
	for i, m := range messages {
		apiMsgs[i] = map[string]string{"role": m.Role, "content": m.Content}
	}

	body, err := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"system":     system,
		"messages":   apiMsgs,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, respBody)
	}

	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}
	return apiResp.Content[0].Text, nil
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

	msgs := []Message{{Role: "user", Content: sql}}

	name, err := c.complete(ctx, c.FastModel, system, msgs, 32)
	if err != nil {
		return "", err
	}
	// Strip any surrounding quotes the model might add
	name = strings.Trim(strings.TrimSpace(name), "\"'`")
	return name, nil
}

// selectTables asks a fast model which tables are relevant to the prompt.
// Falls back to the full schema on any error.
func (c *Client) selectTables(ctx context.Context, schema map[string][]Column, prompt string) map[string][]Column {
	names := make([]string, 0, len(schema))
	for fqn := range schema {
		names = append(names, fqn)
	}

	system := `You select which database tables are relevant to a user's query.
Return ONLY a JSON array of table names (e.g. ["public.users", "public.orders"]).
Include tables that might be needed for JOINs. When in doubt, include the table.`

	msgs := []Message{{Role: "user", Content: fmt.Sprintf("Tables: %s\n\nQuery: %s", strings.Join(names, ", "), prompt)}}

	text, err := c.complete(ctx, c.FastModel, system, msgs, 512)
	if err != nil {
		return schema
	}

	var selected []string
	if err := extractJSON(text, &selected); err != nil || len(selected) == 0 {
		return schema
	}

	filtered := make(map[string][]Column, len(selected))
	for _, name := range selected {
		if cols, ok := schema[name]; ok {
			filtered[name] = cols
		}
	}
	if len(filtered) == 0 {
		return schema
	}
	return filtered
}

// extractJSON tries to unmarshal text as JSON. If that fails, it looks for the
// outermost {...} or [...] substring and tries again.
func extractJSON[T any](text string, dest *T) error {
	if err := json.Unmarshal([]byte(text), dest); err == nil {
		return nil
	}

	for _, pair := range [][2]string{{"{", "}"}, {"[", "]"}} {
		start := strings.Index(text, pair[0])
		end := strings.LastIndex(text, pair[1])
		if start >= 0 && end > start {
			if err := json.Unmarshal([]byte(text[start:end+1]), dest); err == nil {
				return nil
			}
		}
	}
	return fmt.Errorf("no JSON found in response")
}

func buildMessages(history []Message, prompt string) []Message {
	msgs := make([]Message, 0, len(history)+1)
	msgs = append(msgs, history...)
	msgs = append(msgs, Message{Role: "user", Content: prompt})
	return msgs
}

func formatSchema(schema map[string][]Column) string {
	var b strings.Builder
	for fqn, cols := range schema {
		b.WriteString(fqn)
		b.WriteByte('(')
		for i, c := range cols {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%s %s", c.Name, shortType(c.Type))
		}
		b.WriteString(")\n")
	}
	return b.String()
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
