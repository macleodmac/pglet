package service

import (
	"context"

	"github.com/macleodmac/pglet/pkg/client"
)

func (s *Service) Schemas() ([]string, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.Schemas()
}

func (s *Service) Objects() (map[string]*client.SchemaGroup, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.Objects()
}

func (s *Service) TableColumns(table string) ([]client.Column, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.TableColumns(table)
}

func (s *Service) TableRows(ctx context.Context, table string, limit, offset int, sortCol, sortOrd string) (*client.QueryResult, int, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, 0, err
	}
	return cl.TableRows(ctx, table, limit, offset, sortCol, sortOrd)
}

func (s *Service) TableInfo(table string) (*client.TableInfo, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.TableInfo(table)
}

func (s *Service) TableIndexes(table string) ([]client.TableIndex, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.TableIndexes(table)
}

func (s *Service) TableConstraints(table string) ([]client.TableConstraint, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.TableConstraints(table)
}

func (s *Service) FunctionDefinition(schema, name string) (*client.FunctionDefinition, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.FunctionDefinition(schema, name)
}

func (s *Service) TablesStats() (*client.QueryResult, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.TablesStats()
}

func (s *Service) Activity() ([]client.Activity, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.Activity()
}

func (s *Service) ServerSettings() (*client.QueryResult, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.ServerSettings()
}
