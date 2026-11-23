package api

import (
	"errors"
	"net/http"

	"github.com/macleodmac/pglet/internal/service"
)

func (s *Server) Connect(w http.ResponseWriter, r *http.Request) {
	var req ConnectRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	info, err := s.svc.Connect(r.Context(), req.Url)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, ConnectionInfo{
		Host: info.Host, Port: info.Port, User: info.User,
		Database: info.Database, Version: info.Version,
	})
}

func (s *Server) Disconnect(w http.ResponseWriter, r *http.Request) {
	s.svc.Disconnect()
	success := true
	writeJSON(w, http.StatusOK, SuccessResponse{Success: &success})
}

func (s *Server) SwitchDatabase(w http.ResponseWriter, r *http.Request) {
	var req SwitchDBRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	info, err := s.svc.SwitchDatabase(r.Context(), req.Database)
	if errors.Is(err, service.ErrNotConnected) {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, ConnectionInfo{
		Host: info.Host, Port: info.Port, User: info.User,
		Database: info.Database, Version: info.Version,
	})
}

func (s *Server) GetConnectionInfo(w http.ResponseWriter, r *http.Request) {
	info, connected, connecting, err := s.svc.ConnectionInfo(r.Context())
	if !connected {
		c := false
		resp := ConnectionInfo{Connected: &c}
		if connecting {
			resp.Connecting = &connecting
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, ConnectionInfo{
		Host: info.Host, Port: info.Port, User: info.User,
		Database: info.Database, Version: info.Version,
	})
}

func (s *Server) ListDatabases(w http.ResponseWriter, r *http.Request) {
	dbs, err := s.svc.Databases(r.Context())
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrNotConnected) {
			status = http.StatusBadRequest
		}
		writeErr(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, dbs)
}

func (s *Server) GetAppInfo(w http.ResponseWriter, r *http.Request) {
	v, aiEnabled := s.svc.AppInfo()
	writeJSON(w, http.StatusOK, AppInfo{Version: &v, AiEnabled: &aiEnabled})
}

func (s *Server) ListSavedConnections(w http.ResponseWriter, r *http.Request) {
	conns, err := s.svc.SavedConnections()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	result := SavedConnections{Connections: make([]SavedConnection, len(conns))}
	for i, c := range conns {
		result.Connections[i] = SavedConnection{Name: c.Name, Url: c.URL}
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) SaveConnections(w http.ResponseWriter, r *http.Request) {
	var req SavedConnections
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}
	conns := make([]service.SavedConnection, len(req.Connections))
	for i, c := range req.Connections {
		conns[i] = service.SavedConnection{Name: c.Name, URL: c.Url}
	}
	if err := s.svc.SaveConnections(conns); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	success := true
	writeJSON(w, http.StatusOK, SuccessResponse{Success: &success})
}
