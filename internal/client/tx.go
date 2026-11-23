package client

import (
	"context"
	"database/sql"
	"database/sql/driver"
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
	return c.begin(ctx, "BEGIN")
}

// BeginReadOnly starts a transaction in which PostgreSQL rejects any write.
func (c *Client) BeginReadOnly(ctx context.Context) (*Tx, error) {
	return c.begin(ctx, "BEGIN READ ONLY")
}

func (c *Client) begin(ctx context.Context, stmt string) (*Tx, error) {
	conn, err := c.conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, stmt); err != nil {
		conn.Close()
		return nil, err
	}
	return &Tx{conn: conn}, nil
}

// QueryWithContext runs a query inside the transaction.
func (t *Tx) QueryWithContext(ctx context.Context, query string) (*QueryResult, error) {
	return runQuery(ctx, t.conn, query)
}

// Stream runs a query inside the transaction, passing each row to onRow.
func (t *Tx) Stream(ctx context.Context, query string, onHeader func(columns, types []string) error, onRow func([]*string) error) error {
	return streamQuery(ctx, t.conn, query, onHeader, onRow)
}

// RolledBackTo runs fn inside a savepoint and then rolls back to it, so
// nothing fn does survives and an error in fn does not abort the transaction.
func (t *Tx) RolledBackTo(ctx context.Context, fn func() (*QueryResult, error)) (*QueryResult, error) {
	if _, err := t.conn.ExecContext(ctx, "SAVEPOINT pglet_probe"); err != nil {
		return nil, err
	}
	result, err := fn()
	if _, rerr := t.conn.ExecContext(context.Background(), "ROLLBACK TO SAVEPOINT pglet_probe"); rerr != nil && err == nil {
		err = rerr
	}
	return result, err
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
	if err == nil {
		resetSession(t.conn)
	} else {
		t.conn.Raw(func(any) error { return driver.ErrBadConn })
	}
	if cerr := t.conn.Close(); err == nil {
		err = cerr
	}
	return res, err
}
