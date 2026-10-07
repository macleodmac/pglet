package service

import (
	"context"
	"errors"
	"strings"

	"github.com/macleodmac/pglet/internal/client"
	"github.com/macleodmac/pglet/internal/repository"
)

// DefaultMaxRows is the row cap used when a request does not set one.
const DefaultMaxRows = 1000

// track registers ctx as the tab's running query, cancelling any previous
// one, so CancelQuery can stop it. Call the returned func when done.
func (s *Service) track(ctx context.Context, tabID string) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)

	s.queryMu.Lock()
	if prev, ok := s.running[tabID]; ok {
		prev()
	}
	s.running[tabID] = cancel
	s.queryMu.Unlock()

	return ctx, func() {
		s.queryMu.Lock()
		delete(s.running, tabID)
		s.queryMu.Unlock()
		cancel()
	}
}

// RunQuery runs query for the tab, keeping at most maxRows rows (0 keeps all).
func (s *Service) RunQuery(ctx context.Context, tabID, query string, maxRows int) (*client.QueryResult, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}

	ctx, done := s.track(ctx, tabID)
	defer done()

	// When the tab has an open transaction, run the query inside it so all
	// statements share the same session until commit/rollback.
	var result *client.QueryResult
	if tx, ok := s.getTx(tabID); ok {
		result, err = tx.QueryLimited(ctx, query, maxRows)
	} else {
		result, err = cl.QueryLimited(ctx, query, maxRows)
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
	if result.Truncated && result.RowsAffected != nil {
		entry.RowCount = int(*result.RowsAffected)
	}
	if s.Repo != nil {
		s.Repo.AddHistoryEntry(entry)
	}

	return result, nil
}

// ExplainQuery returns the JSON plan for a single statement without running it.
func (s *Service) ExplainQuery(ctx context.Context, tabID, query string) (*client.QueryResult, error) {
	return s.explain(ctx, tabID, "EXPLAIN (FORMAT JSON) "+query)
}

// AnalyzeQuery executes a single statement under EXPLAIN ANALYZE and returns
// the JSON plan with actual timings. The statement's effects are always rolled
// back.
func (s *Service) AnalyzeQuery(ctx context.Context, tabID, query string) (*client.QueryResult, error) {
	return s.explain(ctx, tabID, "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+query)
}

// explain runs an EXPLAIN statement so that it can never change data: inside a
// savepoint when the tab has a transaction (so it sees the tab's uncommitted
// work), otherwise in a throwaway transaction. Both reject multiple statements,
// which would otherwise let "EXPLAIN ANALYZE SELECT 1; DELETE ..." through.
func (s *Service) explain(ctx context.Context, tabID, stmt string) (*client.QueryResult, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}

	ctx, done := s.track(ctx, tabID)
	defer done()

	var result *client.QueryResult
	if tx, ok := s.getTx(tabID); ok {
		result, err = tx.QueryRolledBack(ctx, stmt)
	} else {
		result, err = cl.QueryRolledBack(ctx, stmt)
	}
	if err != nil {
		return nil, &QueryError{Err: explainError(err)}
	}
	return result, nil
}

// explainError rewords the server's multiple-statement error for EXPLAIN.
func explainError(err error) error {
	if strings.Contains(err.Error(), "cannot insert multiple commands into a prepared statement") {
		return errors.New("EXPLAIN works on one statement at a time; select the statement to explain")
	}
	return err
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
