package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func (s *Server) GetSettings(c *gin.Context) {
	settings, err := s.Repo.GetAllSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	settings["ai_api_key_set"] = "false"
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		settings["ai_api_key_set"] = "true"
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
		if k == "ai_api_key" || k == "ai_api_key_set" {
			continue // managed via env var
		}
		if err := s.Repo.SetSetting(c.Request.Context(), k, v); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}
	}
	success := true
	c.JSON(http.StatusOK, SuccessResponse{Success: &success})
}
