package service

import "github.com/macleodmac/pglet/pkg/repository"

func (s *Service) ListHistory(limit, offset int) ([]repository.HistoryEntry, int, error) {
	return s.Repo.ListHistory(limit, offset)
}

func (s *Service) ClearHistory() error {
	return s.Repo.ClearHistory()
}
