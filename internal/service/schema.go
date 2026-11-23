package service

import (
	"context"

	"github.com/macleodmac/pglet/internal/client"
)

func (s *Service) Schemas(ctx context.Context) ([]string, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.Schemas(ctx)
}

func (s *Service) Objects(ctx context.Context) (map[string]*client.SchemaGroup, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.Objects(ctx)
}

func (s *Service) TableColumns(ctx context.Context, table string) ([]client.Column, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.TableColumns(ctx, table)
}

func (s *Service) TableRows(ctx context.Context, table string, limit, offset int, sortCol, sortOrd string) (*client.QueryResult, int, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, 0, err
	}
	return cl.TableRows(ctx, table, limit, offset, sortCol, sortOrd)
}

func (s *Service) TableInfo(ctx context.Context, table string) (*client.TableInfo, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.TableInfo(ctx, table)
}

func (s *Service) TableIndexes(ctx context.Context, table string) ([]client.TableIndex, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.TableIndexes(ctx, table)
}

func (s *Service) TableConstraints(ctx context.Context, table string) ([]client.TableConstraint, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.TableConstraints(ctx, table)
}

func (s *Service) FunctionDefinition(ctx context.Context, schema, name string) (*client.FunctionDefinition, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.FunctionDefinition(ctx, schema, name)
}

func (s *Service) TablesStats(ctx context.Context) (*client.QueryResult, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.TablesStats(ctx)
}

func (s *Service) Activity(ctx context.Context) ([]client.Activity, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.Activity(ctx)
}

func (s *Service) ServerSettings(ctx context.Context) (*client.QueryResult, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.ServerSettings(ctx)
}
