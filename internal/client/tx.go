package client

import (
	"context"
	"database/sql"
	"errors"
)

// Tx is a transaction pinned to a single connection. It is driven with plain
// BEGIN/COMMIT/ROLLBACK rather than sql.Tx so queries can reach the driver for
// command tags.
type Tx struct {
	conn *sql.Conn
}

// Begin starts a new transaction on a dedicated connection from the pool.
func (c *Client) Begin(ctx context.Context) (*Tx, error) {
	conn, err := c.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, "BEGIN"); err != nil {
		conn.Close()
		return nil, err
	}
	return &Tx{conn: conn}, nil
}

// QueryWithContext runs a query inside the transaction.
func (t *Tx) QueryWithContext(ctx context.Context, query string) (*QueryResult, error) {
	return runQuery(ctx, t.conn, query, queryOpts{})
}

// QueryLimited is QueryWithContext with at most maxRows rows kept.
func (t *Tx) QueryLimited(ctx context.Context, query string, maxRows int) (*QueryResult, error) {
	return runQuery(ctx, t.conn, query, queryOpts{maxRows: maxRows})
}

// QueryRolledBack runs a single statement inside a savepoint that is rolled
// back afterwards: its effects are undone and an error in it does not abort
// the surrounding transaction.
func (t *Tx) QueryRolledBack(ctx context.Context, query string) (*QueryResult, error) {
	if _, err := runQuery(ctx, t.conn, "SAVEPOINT pglet_rollback", queryOpts{}); err != nil {
		return nil, err
	}
	res, err := runQuery(ctx, t.conn, query, queryOpts{singleStatement: true})
	// Clean up even when ctx was cancelled.
	bg := context.Background()
	if _, rerr := runQuery(bg, t.conn, "ROLLBACK TO SAVEPOINT pglet_rollback", queryOpts{}); rerr != nil && err == nil {
		err = rerr
	}
	if _, rerr := runQuery(bg, t.conn, "RELEASE SAVEPOINT pglet_rollback", queryOpts{}); rerr != nil && err == nil {
		err = rerr
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

// ErrTxAborted is returned by Commit when the transaction had already failed
// and Postgres rolled it back instead.
var ErrTxAborted = errors.New("transaction was aborted by an earlier error and has been rolled back")

// Commit commits the transaction and releases its connection.
func (t *Tx) Commit() error {
	res, err := t.finish("COMMIT")
	if err == nil && res.CommandTag == "ROLLBACK" {
		return ErrTxAborted
	}
	return err
}

// Rollback aborts the transaction and releases its connection.
func (t *Tx) Rollback() error {
	_, err := t.finish("ROLLBACK")
	return err
}

func (t *Tx) finish(stmt string) (*QueryResult, error) {
	res, err := runQuery(context.Background(), t.conn, stmt, queryOpts{})
	if cerr := t.conn.Close(); err == nil {
		err = cerr
	}
	return res, err
}
