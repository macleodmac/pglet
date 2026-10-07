package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/macleodmac/pglet/internal/client"
	"github.com/macleodmac/pglet/internal/service"
)

func (s *Server) RunQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	maxRows := service.DefaultMaxRows
	if req.MaxRows != nil {
		maxRows = *req.MaxRows
	}
	result, err := s.svc.RunQuery(r.Context(), req.TabId, req.Query, maxRows)
	s.writeQueryResult(w, req.TabId, result, err)
}

// writeQueryResult writes a query outcome. Query execution errors go inside a
// 200 OK result per the API contract; other errors map to an HTTP status.
func (s *Server) writeQueryResult(w http.ResponseWriter, tabID string, result *client.QueryResult, err error) {
	inTx := s.svc.InTransaction(tabID)
	if err != nil {
		var qe *service.QueryError
		if errors.As(err, &qe) {
			errMsg := qe.Error()
			writeJSON(w, http.StatusOK, QueryResult{
				Columns: []string{}, ColumnTypes: []string{},
				Rows: [][]CellValue{}, RowCount: 0, DurationMs: 0,
				Error: &errMsg, InTransaction: &inTx,
			})
			return
		}
		writeErr(w, svcStatus(err), err)
		return
	}

	res := toQueryResult(result)
	res.InTransaction = &inTx
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) BeginTransaction(w http.ResponseWriter, r *http.Request) {
	s.txAction(w, r, func(_ context.Context, tabID string) error {
		return s.svc.BeginTransaction(tabID)
	})
}

func (s *Server) CommitTransaction(w http.ResponseWriter, r *http.Request) {
	s.txAction(w, r, func(_ context.Context, tabID string) error {
		return s.svc.CommitTransaction(tabID)
	})
}

func (s *Server) RollbackTransaction(w http.ResponseWriter, r *http.Request) {
	s.txAction(w, r, func(_ context.Context, tabID string) error {
		return s.svc.RollbackTransaction(tabID)
	})
}

// txAction decodes a TransactionRequest, runs the given transaction operation,
// and returns the resulting transaction status.
func (s *Server) txAction(w http.ResponseWriter, r *http.Request, action func(context.Context, string) error) {
	var req TransactionRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	if err := action(r.Context(), req.TabId); err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}

	writeJSON(w, http.StatusOK, TransactionStatus{InTransaction: s.svc.InTransaction(req.TabId)})
}

func (s *Server) ExplainQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	result, err := s.svc.ExplainQuery(r.Context(), req.TabId, req.Query)
	s.writeQueryResult(w, req.TabId, result, err)
}

func (s *Server) AnalyzeQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	result, err := s.svc.AnalyzeQuery(r.Context(), req.TabId, req.Query)
	s.writeQueryResult(w, req.TabId, result, err)
}

func (s *Server) CancelQuery(w http.ResponseWriter, r *http.Request) {
	var req CancelRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	s.svc.CancelQuery(req.TabId)
	success := true
	writeJSON(w, http.StatusOK, SuccessResponse{Success: &success})
}
