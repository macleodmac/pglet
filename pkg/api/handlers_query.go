package api

import (
	"errors"
	"net/http"

	"github.com/macleodmac/pglet/pkg/service"
)

func (s *Server) RunQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	result, err := s.svc.RunQuery(r.Context(), req.TabId, req.Query)
	if err != nil {
		var qe *service.QueryError
		if errors.As(err, &qe) {
			// Query execution error — return inside 200 OK per API contract
			errMsg := qe.Error()
			writeJSON(w, http.StatusOK, QueryResult{
				Columns: []string{}, ColumnTypes: []string{},
				Rows: [][]CellValue{}, RowCount: 0, DurationMs: 0,
				Error: &errMsg,
			})
			return
		}
		writeErr(w, svcStatus(err), err)
		return
	}

	writeJSON(w, http.StatusOK, QueryResult{
		Columns: result.Columns, ColumnTypes: result.ColumnTypes,
		Rows: toNullableRows(result.Rows), RowCount: result.RowCount,
		DurationMs: result.DurationMs,
	})
}

func (s *Server) ExplainQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	result, err := s.svc.ExplainQuery(r.Context(), req.Query)
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, toQueryResult(result))
}

func (s *Server) AnalyzeQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	result, err := s.svc.AnalyzeQuery(r.Context(), req.Query)
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, toQueryResult(result))
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
