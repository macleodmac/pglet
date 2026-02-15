package service

import "github.com/macleodmac/pglet/pkg/repository"

func (s *Service) ListSavedQueries(database string) ([]repository.SavedQuery, error) {
	return s.Repo.ListSavedQueries(database)
}

func (s *Service) GetSavedQuery(id string) (*repository.SavedQuery, error) {
	return s.Repo.GetSavedQuery(id)
}

func (s *Service) CreateSavedQuery(sq repository.SavedQuery) (*repository.SavedQuery, error) {
	return s.Repo.CreateSavedQuery(sq)
}

func (s *Service) UpdateSavedQuery(sq repository.SavedQuery) error {
	return s.Repo.UpdateSavedQuery(sq)
}

func (s *Service) DeleteSavedQuery(id string) error {
	return s.Repo.DeleteSavedQuery(id)
}
