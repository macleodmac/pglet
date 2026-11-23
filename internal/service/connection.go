package service

import (
	"context"
	"encoding/json"

	"github.com/macleodmac/pglet/internal/client"
)

func (s *Service) Connect(ctx context.Context, url string) (*client.ConnectionInfo, error) {
	cl, err := client.New(url)
	if err != nil {
		return nil, err
	}
	s.rollbackAllTransactions()
	s.SwapClient(cl)
	return cl.Info(ctx)
}

func (s *Service) Disconnect() {
	s.rollbackAllTransactions()
	s.SwapClient(nil)
}

func (s *Service) SwitchDatabase(ctx context.Context, database string) (*client.ConnectionInfo, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	newClient, err := cl.SwitchDatabase(database)
	if err != nil {
		return nil, err
	}
	s.rollbackAllTransactions()
	s.SwapClient(newClient)
	return newClient.Info(ctx)
}

func (s *Service) ConnectionInfo(ctx context.Context) (info *client.ConnectionInfo, connected bool, connecting bool, err error) {
	s.mu.RLock()
	cl := s.client
	conn := s.connecting
	s.mu.RUnlock()

	if cl == nil {
		return nil, false, conn, nil
	}
	info, err = cl.Info(ctx)
	if err != nil {
		return nil, true, false, err
	}
	return info, true, false, nil
}

func (s *Service) Databases(ctx context.Context) ([]string, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.Databases(ctx)
}

type SavedConnection struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (s *Service) SavedConnections() ([]SavedConnection, error) {
	data, err := s.Repo.SavedConnections()
	if err != nil || data == "" {
		return []SavedConnection{}, err
	}
	var conns []SavedConnection
	if err := json.Unmarshal([]byte(data), &conns); err != nil {
		return nil, err
	}
	return conns, nil
}

func (s *Service) SaveConnections(conns []SavedConnection) error {
	data, err := json.Marshal(conns)
	if err != nil {
		return err
	}
	return s.Repo.SaveConnections(string(data))
}
