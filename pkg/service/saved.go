package service

import (
	"context"

	"github.com/macleodmac/pglet/pkg/repository"
)

func (s *Service) ListSavedQueries(ctx context.Context, database string) ([]repository.SavedQuery, error) {
	return s.Repo.ListSavedQueries(ctx, database)
}

func (s *Service) GetSavedQuery(ctx context.Context, id string) (*repository.SavedQuery, error) {
	return s.Repo.GetSavedQuery(ctx, id)
}

func (s *Service) CreateSavedQuery(ctx context.Context, sq repository.SavedQuery) (*repository.SavedQuery, error) {
	return s.Repo.CreateSavedQuery(ctx, sq)
}

func (s *Service) UpdateSavedQuery(ctx context.Context, sq repository.SavedQuery) error {
	return s.Repo.UpdateSavedQuery(ctx, sq)
}

func (s *Service) DeleteSavedQuery(ctx context.Context, id string) error {
	return s.Repo.DeleteSavedQuery(ctx, id)
}
