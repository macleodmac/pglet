package api

import "net/http"

func (s *Server) GetTabState(w http.ResponseWriter, r *http.Request) {
	data, err := s.svc.GetTabState(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, TabState{Data: &data})
}

func (s *Server) SaveTabState(w http.ResponseWriter, r *http.Request) {
	var req TabState
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}
	data := ""
	if req.Data != nil {
		data = *req.Data
	}
	if err := s.svc.SaveTabState(r.Context(), data); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	success := true
	writeJSON(w, http.StatusOK, SuccessResponse{Success: &success})
}
