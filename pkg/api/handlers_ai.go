package api

import (
	"net/http"

	"github.com/macleodmac/pglet/pkg/ai"
)

func (s *Server) AiGenerate(w http.ResponseWriter, r *http.Request) {
	var req AiGenerateRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	var history []ai.Message
	if req.Messages != nil {
		for _, m := range *req.Messages {
			history = append(history, ai.Message{Role: m.Role, Content: m.Content})
		}
	}

	sql, explanation, err := s.svc.GenerateSQL(r.Context(), req.Prompt, history)
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}

	writeJSON(w, http.StatusOK, AiGenerateResponse{Sql: sql, Explanation: explanation})
}

func (s *Server) AiSuggestions(w http.ResponseWriter, r *http.Request) {
	suggestions, err := s.svc.AISuggestions(r.Context())
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, AiSuggestionsResponse{Suggestions: suggestions})
}

func (s *Server) AiTabName(w http.ResponseWriter, r *http.Request) {
	var req AiTabNameRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	name, err := s.svc.AITabName(r.Context(), req.Sql)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, AiTabNameResponse{Name: name})
}
