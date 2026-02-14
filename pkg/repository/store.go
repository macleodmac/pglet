package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitemigration"
	"zombiezen.com/go/sqlite/sqlitex"
)

type Repository struct {
	pool *sqlitemigration.Pool
}

var schema = sqlitemigration.Schema{
	Migrations: migrations,
}

func Open(path string) (*Repository, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create repository dir: %w", err)
	}

	pool := sqlitemigration.NewPool(path, schema, sqlitemigration.Options{
		Flags:    sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL,
		PoolSize: 4,
		PrepareConn: func(conn *sqlite.Conn) error {
			return sqlitex.ExecuteTransient(conn, "PRAGMA busy_timeout = 5000;", nil)
		},
	})

	// Verify the pool works by taking and returning a connection
	conn, err := pool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	pool.Put(conn)

	return &Repository{pool: pool}, nil
}

func (r *Repository) Close() error {
	return r.pool.Close()
}

func (r *Repository) conn(ctx context.Context) (*sqlite.Conn, error) {
	return r.pool.Take(ctx)
}

func (r *Repository) put(conn *sqlite.Conn) {
	r.pool.Put(conn)
}
