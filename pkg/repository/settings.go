package repository

import (
	"context"
	"fmt"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func (r *Repository) GetSetting(ctx context.Context, key string) (string, error) {
	conn, err := r.conn(ctx)
	if err != nil {
		return "", err
	}
	defer r.put(conn)

	var val string
	var found bool
	err = sqlitex.Execute(conn, "SELECT value FROM settings WHERE key = ?", &sqlitex.ExecOptions{
		Args: []any{key},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			val = stmt.GetText("value")
			found = true
			return nil
		},
	})
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("setting not found: %s", key)
	}
	return val, nil
}

func (r *Repository) SetSetting(ctx context.Context, key, value string) error {
	conn, err := r.conn(ctx)
	if err != nil {
		return err
	}
	defer r.put(conn)

	return sqlitex.Execute(conn, "INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", &sqlitex.ExecOptions{
		Args: []any{key, value},
	})
}

func (r *Repository) GetAllSettings(ctx context.Context) (map[string]string, error) {
	conn, err := r.conn(ctx)
	if err != nil {
		return nil, err
	}
	defer r.put(conn)

	result := make(map[string]string)
	err = sqlitex.Execute(conn, "SELECT key, value FROM settings", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			result[stmt.GetText("key")] = stmt.GetText("value")
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
