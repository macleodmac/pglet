package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jm/pglet/pkg/ai"
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
