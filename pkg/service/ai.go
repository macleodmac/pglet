package service

import (
	"context"
	"os"
	"strings"

	"github.com/macleodmac/pglet/pkg/ai"
)

func (s *Service) GenerateSQL(ctx context.Context, prompt string, messages []ai.Message) (sql, explanation string, err error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", "", ErrNoAPIKey
	}

	schema := s.buildSchema()
	aiClient := ai.NewClient(apiKey)
	return aiClient.GenerateSQL(ctx, schema, prompt, messages)
}

func (s *Service) AISuggestions(ctx context.Context) ([]string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, ErrNoAPIKey
	}

	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}

	allCols, err := cl.AllTableColumns()
	if err != nil {
		return nil, err
	}

	tables := make([]string, 0, len(allCols))
	for fqn := range allCols {
		tables = append(tables, fqn)
	}

	aiClient := ai.NewClient(apiKey)
	return aiClient.GenerateSuggestions(ctx, tables)
}

func (s *Service) AITabName(ctx context.Context, sql string) (string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return HeuristicTabName(sql), nil
	}

	aiClient := ai.NewClient(apiKey)
	name, err := aiClient.GenerateTabName(ctx, sql)
	if err != nil {
		return HeuristicTabName(sql), nil
	}
	return name, nil
}

func (s *Service) buildSchema() map[string][]ai.Column {
	schema := make(map[string][]ai.Column)
	cl := s.GetClient()
	if cl == nil {
		return schema
	}
	allCols, err := cl.AllTableColumns()
	if err != nil {
		return schema
	}
	for fqn, cols := range allCols {
		aiCols := make([]ai.Column, len(cols))
		for i, col := range cols {
			aiCols[i] = ai.Column{Name: col.Name, Type: col.Type}
		}
		schema[fqn] = aiCols
	}
	return schema
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
