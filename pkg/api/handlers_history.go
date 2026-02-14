package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) ListHistory(c *gin.Context, params ListHistoryParams) {
	limit, offset := 50, 0
	if params.Limit != nil {
		limit = *params.Limit
	}
	if params.Offset != nil {
		offset = *params.Offset
	}

	entries, total, err := s.Repo.ListHistory(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	result := make([]HistoryEntry, len(entries))
	for i, e := range entries {
		result[i] = HistoryEntry{
			Id: e.ID, Sql: e.SQL, Database: e.Database,
			DurationMs: e.DurationMs, RowCount: e.RowCount,
			Error: e.Error, ExecutedAt: e.ExecutedAt,
		}
	}
	c.JSON(http.StatusOK, HistoryResponse{Entries: result, Total: total})
}

func (s *Server) ClearHistory(c *gin.Context) {
	if err := s.Repo.ClearHistory(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	success := true
	c.JSON(http.StatusOK, SuccessResponse{Success: &success})
}
