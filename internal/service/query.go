package service

import (
	"context"

	"github.com/macleodmac/pglet/internal/client"
	"github.com/macleodmac/pglet/internal/repository"
)

// RunQuery runs a query for a tab, inside the tab's transaction when one is
// open. Otherwise, with safeWrites, a query that writes is held in a new
// transaction for the user to commit or roll back, and held is true.
func (s *Service) RunQuery(ctx context.Context, tabID, query string, safeWrites bool) (result *client.QueryResult, held bool, err error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, false, err
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

	if tx, ok := s.getTx(tabID); ok {
		result, err = tx.QueryWithContext(ctx, query)
	} else if safeWrites {
		result, held, err = s.runGuarded(ctx, cl, tabID, query)
	} else {
		result, err = cl.QueryWithContext(ctx, query)
	}

	entry := repository.HistoryEntry{SQL: query, Database: cl.Database()}
	if err != nil {
		entry.Error = err.Error()
		if s.Repo != nil {
			s.Repo.AddHistoryEntry(entry)
		}
		return nil, false, &QueryError{Err: err}
	}

	entry.DurationMs = result.DurationMs
	entry.RowCount = result.RowCount
	if s.Repo != nil {
		s.Repo.AddHistoryEntry(entry)
	}

	return result, held, nil
}

// runGuarded runs query in a new transaction. It commits straight away if the
// query only read, and otherwise leaves the transaction open on the tab.
func (s *Service) runGuarded(ctx context.Context, cl *client.Client, tabID, query string) (*client.QueryResult, bool, error) {
	tx, err := cl.Begin(ctx)
	if err != nil {
		return nil, false, err
	}

	result, err := tx.QueryWithContext(ctx, query)
	if err != nil {
		tx.Rollback()
		return nil, false, err
	}

	wrote, err := tx.HasWrites(ctx)
	if err != nil {
		tx.Rollback()
		return nil, false, err
	}
	if !wrote {
		return result, false, tx.Commit()
	}

	if !s.storeTx(tabID, tx) {
		tx.Rollback()
		return nil, false, ErrTxConflict
	}
	return result, true, nil
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
