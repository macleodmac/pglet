package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/macleodmac/pglet/pkg/ai"
)

func (s *Server) AiGenerate(c *gin.Context) {
	var req AiGenerateRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	apiKey, _ := s.Repo.GetSetting(c.Request.Context(), "ai_api_key")
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "AI API key not configured. Set it in Settings."})
		return
	}

	// Build schema from connected database.
	schema := make(map[string][]ai.Column)
	if cl := s.getClient(); cl != nil {
		if allCols, err := cl.AllTableColumns(); err == nil {
			for fqn, cols := range allCols {
				aiCols := make([]ai.Column, len(cols))
				for i, col := range cols {
					aiCols[i] = ai.Column{Name: col.Name, Type: col.Type}
				}
				schema[fqn] = aiCols
			}
		}
	}

	// Convert conversation history.
	var history []ai.Message
	if req.Messages != nil {
		for _, m := range *req.Messages {
			history = append(history, ai.Message{Role: m.Role, Content: m.Content})
		}
	}

	aiClient := ai.NewClient(apiKey)
	sql, explanation, err := aiClient.GenerateSQL(c.Request.Context(), schema, req.Prompt, history)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, AiGenerateResponse{Sql: sql, Explanation: explanation})
}

func (s *Server) AiSuggestions(c *gin.Context) {
	apiKey, _ := s.Repo.GetSetting(c.Request.Context(), "ai_api_key")
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "AI API key not configured. Set it in Settings."})
		return
	}

	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Not connected to a database"})
		return
	}

	allCols, err := cl.AllTableColumns()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to fetch schema"})
		return
	}

	tables := make([]string, 0, len(allCols))
	for fqn := range allCols {
		tables = append(tables, fqn)
	}

	aiClient := ai.NewClient(apiKey)
	suggestions, err := aiClient.GenerateSuggestions(c.Request.Context(), tables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, AiSuggestionsResponse{Suggestions: suggestions})
}

func (s *Server) AiTabName(c *gin.Context) {
	var req AiTabNameRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	apiKey, _ := s.Repo.GetSetting(c.Request.Context(), "ai_api_key")
	if apiKey == "" {
		// No API key — fall back to SQL heuristic
		c.JSON(http.StatusOK, AiTabNameResponse{Name: heuristicTabName(req.Sql)})
		return
	}

	aiClient := ai.NewClient(apiKey)
	name, err := aiClient.GenerateTabName(c.Request.Context(), req.Sql)
	if err != nil {
		// AI failed — fall back to heuristic
		c.JSON(http.StatusOK, AiTabNameResponse{Name: heuristicTabName(req.Sql)})
		return
	}

	c.JSON(http.StatusOK, AiTabNameResponse{Name: name})
}

// heuristicTabName extracts a short name from SQL without AI.
// e.g. "SELECT * FROM users WHERE active" → "Select Users"
func heuristicTabName(sql string) string {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return ""
	}

	upper := strings.ToUpper(sql)

	// Detect the verb
	verb := "Query"
	for _, v := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP"} {
		if strings.HasPrefix(upper, v) {
			verb = strings.ToUpper(v[:1]) + strings.ToLower(v[1:])
			break
		}
	}

	// Try to find the main table name
	table := extractTableName(upper)
	if table == "" {
		return verb
	}

	// Title-case the table name
	table = strings.ToUpper(table[:1]) + strings.ToLower(table[1:])
	return verb + " " + table
}

// extractTableName pulls the first table name from common SQL patterns.
func extractTableName(upperSQL string) string {
	// Look for FROM/INTO/UPDATE/TABLE keywords followed by a table name
	for _, kw := range []string{"FROM ", "INTO ", "UPDATE ", "TABLE "} {
		idx := strings.Index(upperSQL, kw)
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(upperSQL[idx+len(kw):])
		// Take the first word (table name), strip schema prefix
		name := strings.FieldsFunc(rest, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '(' || r == ';' || r == ','
		})
		if len(name) == 0 {
			continue
		}
		t := name[0]
		// Strip schema prefix (e.g. PUBLIC.USERS → USERS)
		if dot := strings.LastIndex(t, "."); dot >= 0 {
			t = t[dot+1:]
		}
		return t
	}
	return ""
}
