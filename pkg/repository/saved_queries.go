package repository

import (
	"context"
	"fmt"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type SavedQuery struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	SQL         string `json:"sql"`
	Database    string `json:"database"`
	Tags        string `json:"tags"`
	Shared      bool   `json:"shared"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func scanSavedQuery(stmt *sqlite.Stmt) SavedQuery {
	return SavedQuery{
		ID:          stmt.GetText("id"),
		Title:       stmt.GetText("title"),
		Description: stmt.GetText("description"),
		SQL:         stmt.GetText("sql"),
		Database:    stmt.GetText("database"),
		Tags:        stmt.GetText("tags"),
		Shared:      stmt.GetBool("shared"),
		CreatedAt:   stmt.GetText("created_at"),
		UpdatedAt:   stmt.GetText("updated_at"),
	}
}

func (r *Repository) ListSavedQueries(ctx context.Context, database string) ([]SavedQuery, error) {
	conn, err := r.conn(ctx)
	if err != nil {
		return nil, err
	}
	defer r.put(conn)

	var query string
	var args []any
	if database != "" {
		query = "SELECT id, title, description, sql, database, tags, shared, created_at, updated_at FROM saved_queries WHERE database = ? OR database = '' ORDER BY updated_at DESC"
		args = []any{database}
	} else {
		query = "SELECT id, title, description, sql, database, tags, shared, created_at, updated_at FROM saved_queries ORDER BY updated_at DESC"
	}

	var result []SavedQuery
	err = sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			result = append(result, scanSavedQuery(stmt))
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list saved queries: %w", err)
	}
	if result == nil {
		result = []SavedQuery{}
	}
	return result, nil
}

func (r *Repository) GetSavedQuery(ctx context.Context, id string) (*SavedQuery, error) {
	conn, err := r.conn(ctx)
	if err != nil {
		return nil, err
	}
	defer r.put(conn)

	var q *SavedQuery
	err = sqlitex.Execute(conn, "SELECT id, title, description, sql, database, tags, shared, created_at, updated_at FROM saved_queries WHERE id = ?", &sqlitex.ExecOptions{
		Args: []any{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			sq := scanSavedQuery(stmt)
			q = &sq
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	if q == nil {
		return nil, fmt.Errorf("saved query not found: %s", id)
	}
	return q, nil
}

func (r *Repository) CreateSavedQuery(ctx context.Context, q SavedQuery) (*SavedQuery, error) {
	conn, err := r.conn(ctx)
	if err != nil {
		return nil, err
	}
	defer r.put(conn)

	err = sqlitex.Execute(conn,
		`INSERT INTO saved_queries (title, description, sql, database, tags)
		VALUES (?, ?, ?, ?, ?) RETURNING id, created_at, updated_at`,
		&sqlitex.ExecOptions{
			Args: []any{q.Title, q.Description, q.SQL, q.Database, q.Tags},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				q.ID = stmt.GetText("id")
				q.CreatedAt = stmt.GetText("created_at")
				q.UpdatedAt = stmt.GetText("updated_at")
				return nil
			},
		})
	if err != nil {
		return nil, fmt.Errorf("create saved query: %w", err)
	}
	return &q, nil
}

func (r *Repository) UpdateSavedQuery(ctx context.Context, q SavedQuery) error {
	conn, err := r.conn(ctx)
	if err != nil {
		return err
	}
	defer r.put(conn)

	return sqlitex.Execute(conn,
		`UPDATE saved_queries SET title = ?, description = ?, sql = ?, database = ?, tags = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ?`,
		&sqlitex.ExecOptions{
			Args: []any{q.Title, q.Description, q.SQL, q.Database, q.Tags, q.ID},
		})
}

func (r *Repository) DeleteSavedQuery(ctx context.Context, id string) error {
	conn, err := r.conn(ctx)
	if err != nil {
		return err
	}
	defer r.put(conn)

	return sqlitex.Execute(conn, "DELETE FROM saved_queries WHERE id = ?", &sqlitex.ExecOptions{
		Args: []any{id},
	})
}

func (r *Repository) ListSharedQueries(ctx context.Context) ([]SavedQuery, error) {
	conn, err := r.conn(ctx)
	if err != nil {
		return nil, err
	}
	defer r.put(conn)

	var result []SavedQuery
	err = sqlitex.Execute(conn, "SELECT id, title, description, sql, database, tags, shared, created_at, updated_at FROM saved_queries WHERE shared = 1 ORDER BY title", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			result = append(result, scanSavedQuery(stmt))
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list shared queries: %w", err)
	}
	if result == nil {
		result = []SavedQuery{}
	}
	return result, nil
}

func (r *Repository) UpsertSharedQuery(ctx context.Context, title, sql, description, database, tags string) error {
	conn, err := r.conn(ctx)
	if err != nil {
		return err
	}
	defer r.put(conn)

	// Check if exists by title
	var existingID string
	err = sqlitex.Execute(conn, "SELECT id FROM saved_queries WHERE title = ? AND shared = 1", &sqlitex.ExecOptions{
		Args: []any{title},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			existingID = stmt.GetText("id")
			return nil
		},
	})
	if err != nil {
		return err
	}

	if existingID != "" {
		return sqlitex.Execute(conn,
			`UPDATE saved_queries SET sql = ?, description = ?, database = ?, tags = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ?`,
			&sqlitex.ExecOptions{
				Args: []any{sql, description, database, tags, existingID},
			})
	}

	return sqlitex.Execute(conn,
		`INSERT INTO saved_queries (title, description, sql, database, tags, shared) VALUES (?, ?, ?, ?, ?, 1)`,
		&sqlitex.ExecOptions{
			Args: []any{title, description, sql, database, tags},
		})
}
