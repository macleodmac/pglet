package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	bolt "go.etcd.io/bbolt"
)

// Shared queries live only in their .sql files, which are the source of truth:
// edits and deletes go straight to the file.
const sharedIDPrefix = "shared:"

var ErrNotFound = errors.New("saved query not found")

func isSharedID(id string) bool { return strings.HasPrefix(id, sharedIDPrefix) }

// sharedPath maps a shared query ID to its file, refusing anything that is not
// a plain .sql file name inside sharedDir.
func (r *Repository) sharedPath(id string) (string, error) {
	name := strings.TrimPrefix(id, sharedIDPrefix)
	if r.sharedDir == "" || name == "" || filepath.Base(name) != name || !strings.HasSuffix(name, ".sql") {
		return "", ErrNotFound
	}
	return filepath.Join(r.sharedDir, name), nil
}

func (r *Repository) listSharedQueries() ([]SavedQuery, error) {
	if r.sharedDir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(r.sharedDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var result []SavedQuery
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if q, err := r.getSharedQuery(sharedIDPrefix + entry.Name()); err == nil {
			result = append(result, *q)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Title < result[j].Title })
	return result, nil
}

func (r *Repository) getSharedQuery(id string) (*SavedQuery, error) {
	path, err := r.sharedPath(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	content := string(data)
	if !strings.HasPrefix(content, "-- pglet-query") {
		return nil, ErrNotFound
	}
	q := parseSharedQueryFile(content)
	if q.Title == "" || q.SQL == "" {
		return nil, ErrNotFound
	}
	q.ID = id
	q.Shared = true
	if info, err := os.Stat(path); err == nil {
		q.UpdatedAt = info.ModTime().UTC().Format("2006-01-02T15:04:05Z")
		q.CreatedAt = q.UpdatedAt
	}
	return &q, nil
}

func (r *Repository) updateSharedQuery(q SavedQuery) error {
	path, err := r.sharedPath(q.ID)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return ErrNotFound
	}
	return os.WriteFile(path, []byte(formatSharedQueryFile(q)), 0644)
}

func (r *Repository) deleteSharedQuery(id string) error {
	path, err := r.sharedPath(id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// dropImportedSharedQueries removes copies of shared queries that older
// versions imported into the store; they would otherwise show up twice.
func (r *Repository) dropImportedSharedQueries() error {
	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSavedQueries)
		var stale [][]byte
		b.ForEach(func(k, v []byte) error {
			var q SavedQuery
			if json.Unmarshal(v, &q) == nil && q.Shared {
				stale = append(stale, append([]byte{}, k...))
			}
			return nil
		})
		for _, k := range stale {
			if err := b.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}

func formatSharedQueryFile(q SavedQuery) string {
	var b strings.Builder
	b.WriteString("-- pglet-query\n")
	fmt.Fprintf(&b, "-- @title: %s\n", oneLine(q.Title))
	if q.Description != "" {
		fmt.Fprintf(&b, "-- @description: %s\n", oneLine(q.Description))
	}
	db := q.Database
	if db == "" {
		db = "*"
	}
	fmt.Fprintf(&b, "-- @database: %s\n", oneLine(db))
	if q.Tags != "" {
		fmt.Fprintf(&b, "-- @tags: %s\n", oneLine(q.Tags))
	}
	b.WriteString("\n")
	b.WriteString(q.SQL)
	if !strings.HasSuffix(q.SQL, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

// oneLine keeps a header value from spilling into the SQL body.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func parseSharedQueryFile(content string) SavedQuery {
	var q SavedQuery
	lines := strings.Split(content, "\n")
	headerDone := false
	var sqlLines []string

	for _, line := range lines {
		if headerDone {
			sqlLines = append(sqlLines, line)
			continue
		}
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "" && len(sqlLines) == 0:
			headerDone = true
		case strings.HasPrefix(trimmed, "-- @title:"):
			q.Title = strings.TrimSpace(strings.TrimPrefix(trimmed, "-- @title:"))
		case strings.HasPrefix(trimmed, "-- @description:"):
			q.Description = strings.TrimSpace(strings.TrimPrefix(trimmed, "-- @description:"))
		case strings.HasPrefix(trimmed, "-- @database:"):
			q.Database = strings.TrimSpace(strings.TrimPrefix(trimmed, "-- @database:"))
			if q.Database == "*" {
				q.Database = ""
			}
		case strings.HasPrefix(trimmed, "-- @tags:"):
			q.Tags = strings.TrimSpace(strings.TrimPrefix(trimmed, "-- @tags:"))
		case !strings.HasPrefix(trimmed, "--"):
			headerDone = true
			sqlLines = append(sqlLines, line)
		}
	}

	q.SQL = strings.TrimSpace(strings.Join(sqlLines, "\n"))
	return q
}
