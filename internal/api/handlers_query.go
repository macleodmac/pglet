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

	safeWrites := req.SafeWrites != nil && *req.SafeWrites
	result, held, err := s.svc.RunQuery(r.Context(), req.TabId, req.Query, safeWrites)
	inTx := s.svc.InTransaction(req.TabId)
	if err != nil {
		writeQueryErr(w, err, &inTx)
		return
	}

	res := toQueryResult(result)
	res.InTransaction = &inTx
	res.HeldForReview = &held
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) BeginTransaction(w http.ResponseWriter, r *http.Request) {
	s.txAction(w, r, s.svc.BeginTransaction)
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
	s.explain(w, r, false)
}

func (s *Server) AnalyzeQuery(w http.ResponseWriter, r *http.Request) {
	s.explain(w, r, true)
}

func (s *Server) explain(w http.ResponseWriter, r *http.Request, analyze bool) {
	var req QueryRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	result, err := s.svc.ExplainQuery(r.Context(), req.TabId, req.Query, analyze)
	inTx := s.svc.InTransaction(req.TabId)
	if err != nil {
		writeQueryErr(w, err, &inTx)
		return
	}
	res := toQueryResult(result)
	res.InTransaction = &inTx
	writeJSON(w, http.StatusOK, res)
}

// writeQueryErr reports a query execution error inside a 200 result, which is
// how the UI shows SQL errors; other errors get an HTTP error status.
func writeQueryErr(w http.ResponseWriter, err error, inTx *bool) {
	var qe *service.QueryError
	if !errors.As(err, &qe) {
		writeErr(w, svcStatus(err), err)
		return
	}
	errMsg := qe.Error()
	res := QueryResult{
		Columns: []string{}, ColumnTypes: []string{},
		Rows: [][]*CellValue{}, Error: &errMsg, InTransaction: inTx,
	}
	if code := client.SQLState(qe.Err); code != "" {
		res.ErrorCode = &code
	}
	writeJSON(w, http.StatusOK, res)
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

func (s *Server) ListOpenTransactions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, OpenTransactions{TabIds: s.svc.OpenTransactions()})
}
