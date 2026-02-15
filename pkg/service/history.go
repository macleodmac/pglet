package service

import (
	"context"

	"github.com/macleodmac/pglet/pkg/repository"
)

func (s *Service) ListHistory(ctx context.Context, limit, offset int) ([]repository.HistoryEntry, int, error) {
	return s.Repo.ListHistory(ctx, limit, offset)
}

func (s *Service) ClearHistory(ctx context.Context) error {
	return s.Repo.ClearHistory(ctx)
}
