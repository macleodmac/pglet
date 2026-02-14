package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) GetSettings(c *gin.Context) {
	settings, err := s.Repo.GetAllSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	if key, ok := settings["ai_api_key"]; ok && len(key) > 8 {
		settings["ai_api_key"] = key[:4] + "..." + key[len(key)-4:]
	}
	c.JSON(http.StatusOK, settings)
}

func (s *Server) UpdateSettings(c *gin.Context) {
	var settings map[string]string
	if err := c.BindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}
	for k, v := range settings {
		if err := s.Repo.SetSetting(c.Request.Context(), k, v); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}
	}
	success := true
	c.JSON(http.StatusOK, SuccessResponse{Success: &success})
}
