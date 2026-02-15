package api

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/macleodmac/pglet/pkg/repository"
)

var (
	runningQueries   = map[string]context.CancelFunc{}
	runningQueriesMu sync.Mutex
)

func (s *Server) RunQuery(c *gin.Context) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}

	var req QueryRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	ctx, cancel := context.WithCancel(c.Request.Context())

	runningQueriesMu.Lock()
	if prev, ok := runningQueries[req.TabId]; ok {
		prev()
	}
	runningQueries[req.TabId] = cancel
	runningQueriesMu.Unlock()

	defer func() {
		runningQueriesMu.Lock()
		delete(runningQueries, req.TabId)
		runningQueriesMu.Unlock()
		cancel()
	}()

	result, err := cl.QueryWithContext(ctx, req.Query)

	entry := repository.HistoryEntry{SQL: req.Query, Database: cl.Database()}
	if err != nil {
		entry.Error = err.Error()
		if s.Repo != nil {
			s.Repo.AddHistoryEntry(c.Request.Context(), entry)
		}
		errMsg := err.Error()
		c.JSON(http.StatusOK, QueryResult{
			Columns: []string{}, ColumnTypes: []string{},
			Rows: [][]CellValue{}, RowCount: 0, DurationMs: 0,
			Error: &errMsg,
		})
		return
	}

	entry.DurationMs = result.DurationMs
	entry.RowCount = result.RowCount
	if s.Repo != nil {
		s.Repo.AddHistoryEntry(c.Request.Context(), entry)
	}

	c.JSON(http.StatusOK, QueryResult{
		Columns: result.Columns, ColumnTypes: result.ColumnTypes,
		Rows: toNullableRows(result.Rows), RowCount: result.RowCount,
		DurationMs: result.DurationMs,
	})
}

func (s *Server) ExplainQuery(c *gin.Context) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}
	var req QueryRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	result, err := cl.QueryWithContext(c.Request.Context(), "EXPLAIN "+req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, toQueryResult(result))
}

func (s *Server) AnalyzeQuery(c *gin.Context) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}
	var req QueryRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	result, err := cl.QueryWithContext(c.Request.Context(), "EXPLAIN ANALYZE "+req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, toQueryResult(result))
}

func (s *Server) CancelQuery(c *gin.Context) {
	var req CancelRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	runningQueriesMu.Lock()
	if cancel, ok := runningQueries[req.TabId]; ok {
		cancel()
	}
	runningQueriesMu.Unlock()

	success := true
	c.JSON(http.StatusOK, SuccessResponse{Success: &success})
}
