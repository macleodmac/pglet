package service

import "github.com/macleodmac/pglet/pkg/client"

func (s *Service) Connect(url string) (*client.ConnectionInfo, error) {
	cl, err := client.New(url)
	if err != nil {
		return nil, err
	}
	s.SwapClient(cl)
	info, err := cl.Info()
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (s *Service) Disconnect() {
	s.rollbackAllTransactions()
	s.SwapClient(nil)
}

func (s *Service) SwitchDatabase(database string) (*client.ConnectionInfo, error) {
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
	info, err := newClient.Info()
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (s *Service) ConnectionInfo() (info *client.ConnectionInfo, connected bool, connecting bool, err error) {
	s.mu.RLock()
	cl := s.client
	conn := s.connecting
	s.mu.RUnlock()

	if cl == nil {
		return nil, false, conn, nil
	}
	info, err = cl.Info()
	if err != nil {
		return nil, true, false, err
	}
	return info, true, false, nil
}

func (s *Service) Databases() ([]string, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.Databases()
}
