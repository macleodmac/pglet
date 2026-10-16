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
	return runQuery(ctx, t.conn, query)
}

// HasWrites reports whether the transaction has modified data or taken row
// locks: PostgreSQL only assigns a transaction ID once it needs one.
func (t *Tx) HasWrites(ctx context.Context) (bool, error) {
	var assigned bool
	err := t.conn.QueryRowContext(ctx, "SELECT txid_current_if_assigned() IS NOT NULL").Scan(&assigned)
	return assigned, err
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
	res, err := runQuery(context.Background(), t.conn, stmt)
	if cerr := t.conn.Close(); err == nil {
		err = cerr
	}
	return res, err
}
