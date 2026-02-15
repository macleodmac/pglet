package api

import "net/http"

func (s *Server) GetTabState(w http.ResponseWriter, r *http.Request) {
	data, err := s.svc.GetTabState()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, TabState{Data: &data})
}

func (s *Server) SaveTabState(w http.ResponseWriter, r *http.Request) {
	var req TabState
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}
	data := ""
	if req.Data != nil {
		data = *req.Data
	}
	if err := s.svc.SaveTabState(data); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	success := true
	writeJSON(w, http.StatusOK, SuccessResponse{Success: &success})
}
