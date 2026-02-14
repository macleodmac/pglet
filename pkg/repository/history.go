package repository

import (
	"context"
	"fmt"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type HistoryEntry struct {
	ID         int    `json:"id"`
	SQL        string `json:"sql"`
	Database   string `json:"database"`
	DurationMs int64  `json:"duration_ms"`
	RowCount   int    `json:"row_count"`
	Error      string `json:"error"`
	ExecutedAt string `json:"executed_at"`
}

func (r *Repository) AddHistoryEntry(ctx context.Context, e HistoryEntry) error {
	conn, err := r.conn(ctx)
	if err != nil {
		return err
	}
	defer r.put(conn)

	return sqlitex.Execute(conn,
		`INSERT INTO query_history (sql, database, duration_ms, row_count, error) VALUES (?, ?, ?, ?, ?)`,
		&sqlitex.ExecOptions{
			Args: []any{e.SQL, e.Database, e.DurationMs, e.RowCount, e.Error},
		})
}

func (r *Repository) ListHistory(ctx context.Context, limit, offset int) ([]HistoryEntry, int, error) {
	conn, err := r.conn(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer r.put(conn)

	var total int
	err = sqlitex.Execute(conn, "SELECT COUNT(*) FROM query_history", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			total = stmt.ColumnInt(0)
			return nil
		},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count history: %w", err)
	}

	var result []HistoryEntry
	err = sqlitex.Execute(conn,
		"SELECT id, sql, database, duration_ms, row_count, error, executed_at FROM query_history ORDER BY executed_at DESC LIMIT ? OFFSET ?",
		&sqlitex.ExecOptions{
			Args: []any{limit, offset},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				result = append(result, HistoryEntry{
					ID:         stmt.ColumnInt(0),
					SQL:        stmt.GetText("sql"),
					Database:   stmt.GetText("database"),
					DurationMs: stmt.GetInt64("duration_ms"),
					RowCount:   stmt.ColumnInt(4),
					Error:      stmt.GetText("error"),
					ExecutedAt: stmt.GetText("executed_at"),
				})
				return nil
			},
		})
	if err != nil {
		return nil, 0, fmt.Errorf("list history: %w", err)
	}
	if result == nil {
		result = []HistoryEntry{}
	}
	return result, total, nil
}

func (r *Repository) ClearHistory(ctx context.Context) error {
	conn, err := r.conn(ctx)
	if err != nil {
		return err
	}
	defer r.put(conn)

	return sqlitex.Execute(conn, "DELETE FROM query_history", nil)
}
