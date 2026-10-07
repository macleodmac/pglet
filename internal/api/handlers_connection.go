package api

import (
	"errors"
	"net/http"

	"github.com/macleodmac/pglet/internal/client"
	"github.com/macleodmac/pglet/internal/service"
)

func (s *Server) Connect(w http.ResponseWriter, r *http.Request) {
	var req ConnectRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	opts := client.Options{}
	if req.ReadOnly != nil {
		opts.ReadOnly = *req.ReadOnly
	}
	if req.StatementTimeoutMs != nil {
		opts.StatementTimeoutMs = *req.StatementTimeoutMs
	}
	info, err := s.svc.Connect(req.Url, opts)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, toConnectionInfo(info))
}

func (s *Server) SetConnectionOptions(w http.ResponseWriter, r *http.Request) {
	var req ConnectionOptions
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}
	info, err := s.svc.SetConnectionOptions(client.Options{
		ReadOnly: req.ReadOnly, StatementTimeoutMs: req.StatementTimeoutMs,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toConnectionInfo(info))
}

func toConnectionInfo(info *client.ConnectionInfo) ConnectionInfo {
	return ConnectionInfo{
		Host: info.Host, Port: info.Port, User: info.User,
		Database: info.Database, Version: info.Version,
		ReadOnly: &info.Options.ReadOnly, StatementTimeoutMs: &info.Options.StatementTimeoutMs,
	}
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

	info, err := s.svc.SwitchDatabase(req.Database)
	if errors.Is(err, service.ErrNotConnected) {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, toConnectionInfo(info))
}

func (s *Server) GetConnectionInfo(w http.ResponseWriter, r *http.Request) {
	info, connected, connecting, err := s.svc.ConnectionInfo()
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
	writeJSON(w, http.StatusOK, toConnectionInfo(info))
}

func (s *Server) ListDatabases(w http.ResponseWriter, r *http.Request) {
	dbs, err := s.svc.Databases()
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
