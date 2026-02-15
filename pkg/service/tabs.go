package service

import "context"

func (s *Service) GetTabState(ctx context.Context) (string, error) {
	return s.Repo.GetTabState(ctx)
}

func (s *Service) SaveTabState(ctx context.Context, data string) error {
	return s.Repo.SaveTabState(ctx, data)
}
