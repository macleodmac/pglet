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
	s.SwapClient(newClient)
	info, err := newClient.Info()
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (s *Service) ConnectionInfo() (*client.ConnectionInfo, bool, error) {
	cl := s.GetClient()
	if cl == nil {
		return nil, false, nil
	}
	info, err := cl.Info()
	if err != nil {
		return nil, true, err
	}
	return info, true, nil
}

func (s *Service) Databases() ([]string, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.Databases()
}
