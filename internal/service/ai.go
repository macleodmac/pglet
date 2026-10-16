package service

import (
	"context"
	"os"
	"strings"

	"github.com/macleodmac/pglet/internal/ai"
)

func (s *Service) GenerateSQL(ctx context.Context, prompt string, messages []ai.Message) (sql, explanation string, err error) {
	aiClient, err := s.aiClient()
	if err != nil {
		return "", "", err
	}
	schema, err := s.schemaContext(ctx)
	if err != nil {
		return "", "", err
	}
	return aiClient.GenerateSQL(ctx, schema, prompt, messages)
}

func (s *Service) FixSQL(ctx context.Context, query, errMsg string) (sql, explanation string, err error) {
	aiClient, err := s.aiClient()
	if err != nil {
		return "", "", err
	}
	schema, err := s.schemaContext(ctx)
	if err != nil {
		return "", "", err
	}
	return aiClient.FixSQL(ctx, schema, query, errMsg)
}

func (s *Service) AISuggestions(ctx context.Context) ([]string, error) {
	aiClient, err := s.aiClient()
	if err != nil {
		return nil, err
	}

	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}

	rels, err := cl.Relations(ctx)
	if err != nil {
		return nil, err
	}

	tables := make([]string, len(rels))
	for i, r := range rels {
		tables[i] = r.Schema + "." + r.Name
	}
	return aiClient.GenerateSuggestions(ctx, tables)
}

func (s *Service) AITabName(ctx context.Context, sql string) (string, error) {
	aiClient, err := s.aiClient()
	if err != nil {
		return HeuristicTabName(sql), nil
	}

	name, err := aiClient.GenerateTabName(ctx, sql)
	if err != nil {
		return HeuristicTabName(sql), nil
	}
	return name, nil
}

func (s *Service) aiClient() (*ai.Client, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, ErrNoAPIKey
	}
	return ai.NewClient(apiKey), nil
}

// schemaContext describes the connected database for prompts. It is empty
// when not connected, so generation still works without a schema.
func (s *Service) schemaContext(ctx context.Context) ([]ai.Table, error) {
	cl := s.GetClient()
	if cl == nil {
		return nil, nil
	}
	rels, err := cl.Relations(ctx)
	if err != nil {
		return nil, err
	}

	tables := make([]ai.Table, len(rels))
	for i, r := range rels {
		cols := make([]ai.Column, len(r.Columns))
		for j, c := range r.Columns {
			cols[j] = ai.Column{
				Name:       c.Name,
				Type:       c.Type,
				PrimaryKey: c.IsPrimaryKey,
				References: c.References,
				EnumValues: c.EnumValues,
				Comment:    c.Comment,
			}
		}
		tables[i] = ai.Table{Name: r.Schema + "." + r.Name, Kind: r.Kind, Comment: r.Comment, Columns: cols}
	}
	return tables, nil
}

// HeuristicTabName extracts a short name from SQL without AI.
func HeuristicTabName(sql string) string {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return ""
	}

	upper := strings.ToUpper(sql)

	verb := "Query"
	for _, v := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP"} {
		if strings.HasPrefix(upper, v) {
			verb = strings.ToUpper(v[:1]) + strings.ToLower(v[1:])
			break
		}
	}

	table := extractTableName(upper)
	if table == "" {
		return verb
	}

	table = strings.ToUpper(table[:1]) + strings.ToLower(table[1:])
	return verb + " " + table
}

func extractTableName(upperSQL string) string {
	for _, kw := range []string{"FROM ", "INTO ", "UPDATE ", "TABLE "} {
		idx := strings.Index(upperSQL, kw)
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(upperSQL[idx+len(kw):])
		name := strings.FieldsFunc(rest, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '(' || r == ';' || r == ','
		})
		if len(name) == 0 {
			continue
		}
		t := name[0]
		if dot := strings.LastIndex(t, "."); dot >= 0 {
			t = t[dot+1:]
		}
		return t
	}
	return ""
}
