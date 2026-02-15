package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/macleodmac/pglet/pkg/client"
)

func (s *Server) Connect(c *gin.Context) {
	var req ConnectRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	cl, err := client.New(req.Url)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	s.swapClient(cl)
	info, err := cl.Info()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ConnectionInfo{
		Host: info.Host, Port: info.Port, User: info.User,
		Database: info.Database, Version: info.Version,
	})
}

func (s *Server) Disconnect(c *gin.Context) {
	s.swapClient(nil)
	success := true
	c.JSON(http.StatusOK, SuccessResponse{Success: &success})
}

func (s *Server) SwitchDatabase(c *gin.Context) {
	var req SwitchDBRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}

	newClient, err := cl.SwitchDatabase(req.Database)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	s.swapClient(newClient)

	info, err := newClient.Info()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, ConnectionInfo{
		Host: info.Host, Port: info.Port, User: info.User,
		Database: info.Database, Version: info.Version,
	})
}

func (s *Server) GetConnectionInfo(c *gin.Context) {
	cl := s.getClient()
	if cl == nil {
		connected := false
		c.JSON(http.StatusOK, ConnectionInfo{Connected: &connected})
		return
	}
	info, err := cl.Info()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, ConnectionInfo{
		Host: info.Host, Port: info.Port, User: info.User,
		Database: info.Database, Version: info.Version,
	})
}

func (s *Server) ListDatabases(c *gin.Context) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}
	dbs, err := cl.Databases()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, dbs)
}

// AppVersion is set from main at startup.
var AppVersion = "dev"

func (s *Server) GetAppInfo(c *gin.Context) {
	v := AppVersion
	c.JSON(http.StatusOK, AppInfo{Version: &v})
}
