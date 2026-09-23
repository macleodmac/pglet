package service

import (
	"context"

	"github.com/macleodmac/pglet/internal/client"
	"github.com/macleodmac/pglet/internal/repository"
)

func (s *Service) RunQuery(ctx context.Context, tabID, query string) (*client.QueryResult, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	s.queryMu.Lock()
	if prev, ok := s.running[tabID]; ok {
		prev()
	}
	s.running[tabID] = cancel
	s.queryMu.Unlock()

	defer func() {
		s.queryMu.Lock()
		delete(s.running, tabID)
		s.queryMu.Unlock()
		cancel()
	}()

	// When the tab has an open transaction, run the query inside it so all
	// statements share the same session until commit/rollback.
	var result *client.QueryResult
	if tx, ok := s.getTx(tabID); ok {
		result, err = tx.QueryWithContext(ctx, query)
	} else {
		result, err = cl.QueryWithContext(ctx, query)
	}

	entry := repository.HistoryEntry{SQL: query, Database: cl.Database()}
	if err != nil {
		entry.Error = err.Error()
		if s.Repo != nil {
			s.Repo.AddHistoryEntry(entry)
		}
		return nil, &QueryError{Err: err}
	}

	entry.DurationMs = result.DurationMs
	entry.RowCount = result.RowCount
	if s.Repo != nil {
		s.Repo.AddHistoryEntry(entry)
	}

	return result, nil
}

func (s *Service) ExplainQuery(ctx context.Context, query string) (*client.QueryResult, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.QueryWithContext(ctx, "EXPLAIN "+query)
}

func (s *Service) AnalyzeQuery(ctx context.Context, query string) (*client.QueryResult, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.QueryWithContext(ctx, "EXPLAIN ANALYZE "+query)
}

func (s *Service) CancelQuery(tabID string) {
	s.queryMu.Lock()
	if cancel, ok := s.running[tabID]; ok {
		cancel()
	}
	s.queryMu.Unlock()
}

func (s *Service) ExportQuery(ctx context.Context, query string) (*client.QueryResult, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	return cl.QueryWithContext(ctx, query)
}
