package api

import (
	"net/http"

	"github.com/macleodmac/pglet/pkg/repository"
)

func (s *Server) ListSavedQueries(w http.ResponseWriter, r *http.Request, params ListSavedQueriesParams) {
	database := ""
	if params.Database != nil {
		database = *params.Database
	}
	queries, err := s.svc.ListSavedQueries(r.Context(), database)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	result := make([]SavedQuery, len(queries))
	for i, q := range queries {
		result[i] = repoToSavedQuery(q)
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) GetSavedQuery(w http.ResponseWriter, r *http.Request, id string) {
	q, err := s.svc.GetSavedQuery(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "query not found"})
		return
	}
	writeJSON(w, http.StatusOK, repoToSavedQuery(*q))
}

func (s *Server) CreateSavedQuery(w http.ResponseWriter, r *http.Request) {
	var req SavedQueryInput
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}
	sq := repository.SavedQuery{Title: req.Title, SQL: req.Sql}
	if req.Description != nil {
		sq.Description = *req.Description
	}
	if req.Database != nil {
		sq.Database = *req.Database
	}
	if req.Tags != nil {
		sq.Tags = *req.Tags
	}

	created, err := s.svc.CreateSavedQuery(r.Context(), sq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, repoToSavedQuery(*created))
}

func (s *Server) UpdateSavedQuery(w http.ResponseWriter, r *http.Request, id string) {
	var req SavedQueryInput
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}
	sq := repository.SavedQuery{ID: id, Title: req.Title, SQL: req.Sql}
	if req.Description != nil {
		sq.Description = *req.Description
	}
	if req.Database != nil {
		sq.Database = *req.Database
	}
	if req.Tags != nil {
		sq.Tags = *req.Tags
	}

	if err := s.svc.UpdateSavedQuery(r.Context(), sq); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	success := true
	writeJSON(w, http.StatusOK, SuccessResponse{Success: &success})
}

func (s *Server) DeleteSavedQuery(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.svc.DeleteSavedQuery(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	success := true
	writeJSON(w, http.StatusOK, SuccessResponse{Success: &success})
}

func repoToSavedQuery(q repository.SavedQuery) SavedQuery {
	return SavedQuery{
		Id: q.ID, Title: q.Title, Description: q.Description,
		Sql: q.SQL, Database: q.Database, Tags: q.Tags,
		Shared: q.Shared, CreatedAt: q.CreatedAt, UpdatedAt: q.UpdatedAt,
	}
}
