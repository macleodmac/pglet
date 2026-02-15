package service

func (s *Service) GetTabState() (string, error) {
	return s.Repo.GetTabState()
}

func (s *Service) SaveTabState(data string) error {
	return s.Repo.SaveTabState(data)
}
