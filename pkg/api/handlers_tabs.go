package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) GetTabState(c *gin.Context) {
	data, err := s.Repo.GetTabState(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, TabState{Data: &data})
}

func (s *Server) SaveTabState(c *gin.Context) {
	var req TabState
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}
	data := ""
	if req.Data != nil {
		data = *req.Data
	}
	if err := s.Repo.SaveTabState(c.Request.Context(), data); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	success := true
	c.JSON(http.StatusOK, SuccessResponse{Success: &success})
}
