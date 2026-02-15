package api

import "net/http"

func (s *Server) ListHistory(w http.ResponseWriter, r *http.Request, params ListHistoryParams) {
	limit, offset := 50, 0
	if params.Limit != nil {
		limit = *params.Limit
	}
	if params.Offset != nil {
		offset = *params.Offset
	}

	entries, total, err := s.svc.ListHistory(limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
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
	writeJSON(w, http.StatusOK, HistoryResponse{Entries: result, Total: total})
}

func (s *Server) ClearHistory(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.ClearHistory(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	success := true
	writeJSON(w, http.StatusOK, SuccessResponse{Success: &success})
}
