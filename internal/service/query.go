package service

import (
	"context"
	"errors"

	"github.com/macleodmac/pglet/internal/client"
	"github.com/macleodmac/pglet/internal/repository"
)

var (
	ErrTxControl    = errors.New("BEGIN, COMMIT and ROLLBACK can't be run here; use the Begin, Commit and Rollback buttons")
	ErrQueryRunning = errors.New("a query is already running in this tab")
)

// RunQuery runs a query for a tab, inside the tab's transaction when one is
// open. Otherwise, with safeWrites, a query that writes is held in a new
// transaction for the user to commit or roll back, and held is true.
func (s *Service) RunQuery(ctx context.Context, tabID, query string, safeWrites bool) (result *client.QueryResult, held bool, err error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, false, err
	}

	ctx, done, err := s.track(ctx, tabID)
	if err != nil {
		return nil, false, err
	}
	defer done()

	tx, inTx := s.getTx(tabID)
	switch {
	case (inTx || safeWrites) && hasTransactionControl(query):
		err = ErrTxControl
	case inTx:
		result, err = tx.QueryWithContext(ctx, query)
	case safeWrites:
		result, held, err = s.runGuarded(ctx, cl, tabID, query)
	default:
		result, err = cl.QueryWithContext(ctx, query)
	}

	entry := repository.HistoryEntry{SQL: query, Database: cl.Database()}
	if err != nil {
		entry.Error = err.Error()
		s.addHistory(entry)
		return nil, false, &QueryError{Err: err}
	}

	entry.DurationMs = result.DurationMs
	entry.RowCount = result.RowCount
	s.addHistory(entry)
	return result, held, nil
}

func (s *Service) addHistory(entry repository.HistoryEntry) {
	if s.Repo != nil {
		s.Repo.AddHistoryEntry(entry)
	}
}

// track registers ctx as the tab's running operation so CancelQuery can stop
// it. A tab runs one operation at a time: a second one would either queue
// behind the first on the tab's connection or, if it cancelled the first,
// abort the tab's transaction.
func (s *Service) track(ctx context.Context, tabID string) (context.Context, func(), error) {
	s.queryMu.Lock()
	if _, busy := s.running[tabID]; busy {
		s.queryMu.Unlock()
		return nil, nil, ErrQueryRunning
	}
	ctx, cancel := context.WithCancel(ctx)
	op := &runningOp{cancel: cancel}
	s.running[tabID] = op
	s.queryMu.Unlock()

	return ctx, func() {
		s.queryMu.Lock()
		if s.running[tabID] == op {
			delete(s.running, tabID)
		}
		s.queryMu.Unlock()
		cancel()
	}, nil
}

type runningOp struct {
	cancel context.CancelFunc
}

func hasTransactionControl(query string) bool {
	for _, st := range client.SplitStatements(query) {
		if st.IsTransactionControl() {
			return true
		}
	}
	return false
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

// ExplainQuery shows the plan for a single statement. With analyze the
// statement really runs, so it runs inside a savepoint (in the tab's
// transaction) or a transaction that is always rolled back.
func (s *Service) ExplainQuery(ctx context.Context, tabID, query string, analyze bool) (*client.QueryResult, error) {
	cl, err := s.requireClient()
	if err != nil {
		return nil, err
	}
	st, err := client.SingleStatement(query)
	if err != nil {
		return nil, &QueryError{Err: err}
	}
	if st.IsTransactionControl() {
		return nil, &QueryError{Err: ErrTxControl}
	}

	ctx, done, err := s.track(ctx, tabID)
	if err != nil {
		return nil, err
	}
	defer done()

	stmt := "EXPLAIN " + st.SQL
	if analyze {
		stmt = "EXPLAIN ANALYZE " + st.SQL
	}

	var result *client.QueryResult
	if tx, ok := s.getTx(tabID); ok {
		result, err = tx.RolledBackTo(ctx, func() (*client.QueryResult, error) {
			return tx.QueryWithContext(ctx, stmt)
		})
	} else {
		var tx *client.Tx
		tx, err = cl.Begin(ctx)
		if err != nil {
			return nil, err
		}
		result, err = tx.QueryWithContext(ctx, stmt)
		tx.Rollback()
	}
	if err != nil {
		return nil, &QueryError{Err: err}
	}
	return result, nil
}

func (s *Service) CancelQuery(tabID string) {
	s.queryMu.Lock()
	if op, ok := s.running[tabID]; ok {
		op.cancel()
	}
	s.queryMu.Unlock()
}

// ExportQuery streams the full result of a single read-only statement. It runs
// in a READ ONLY transaction, so a statement that writes fails instead of
// changing data.
func (s *Service) ExportQuery(ctx context.Context, query string, onHeader func(columns, types []string) error, onRow func([]*string) error) error {
	cl, err := s.requireClient()
	if err != nil {
		return err
	}
	st, err := client.SingleStatement(query)
	if err != nil {
		return &QueryError{Err: err}
	}
	if st.IsTransactionControl() {
		return &QueryError{Err: ErrTxControl}
	}
	tx, err := cl.BeginReadOnly(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := tx.Stream(ctx, st.SQL, onHeader, onRow); err != nil {
		return &QueryError{Err: err}
	}
	return nil
}
