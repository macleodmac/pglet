package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jm/pglet/pkg/repository"
)

func (s *Server) ListSavedQueries(c *gin.Context, params ListSavedQueriesParams) {
	database := ""
	if params.Database != nil {
		database = *params.Database
	}
	queries, err := s.Repo.ListSavedQueries(c.Request.Context(), database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	result := make([]SavedQuery, len(queries))
	for i, q := range queries {
		result[i] = repoToSavedQuery(q)
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) GetSavedQuery(c *gin.Context, id string) {
	q, err := s.Repo.GetSavedQuery(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "query not found"})
		return
	}
	c.JSON(http.StatusOK, repoToSavedQuery(*q))
}

func (s *Server) CreateSavedQuery(c *gin.Context) {
	var req SavedQueryInput
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
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

	created, err := s.Repo.CreateSavedQuery(c.Request.Context(), sq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, repoToSavedQuery(*created))
}

func (s *Server) UpdateSavedQuery(c *gin.Context, id string) {
	var req SavedQueryInput
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
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

	if err := s.Repo.UpdateSavedQuery(c.Request.Context(), sq); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	success := true
	c.JSON(http.StatusOK, SuccessResponse{Success: &success})
}

func (s *Server) DeleteSavedQuery(c *gin.Context, id string) {
	if err := s.Repo.DeleteSavedQuery(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	success := true
	c.JSON(http.StatusOK, SuccessResponse{Success: &success})
}

func repoToSavedQuery(q repository.SavedQuery) SavedQuery {
	return SavedQuery{
		Id: q.ID, Title: q.Title, Description: q.Description,
		Sql: q.SQL, Database: q.Database, Tags: q.Tags,
		Shared: q.Shared, CreatedAt: q.CreatedAt, UpdatedAt: q.UpdatedAt,
	}
}
