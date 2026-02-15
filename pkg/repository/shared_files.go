package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExportSharedQueries writes shared queries to .pglet/queries/*.sql files
func (r *Repository) ExportSharedQueries(dir string) error {
	queries, err := r.ListSharedQueries()
	if err != nil {
		return err
	}

	queriesDir := filepath.Join(dir, "queries")
	if err := os.MkdirAll(queriesDir, 0755); err != nil {
		return fmt.Errorf("create queries dir: %w", err)
	}

	for _, q := range queries {
		filename := sanitizeFilename(q.Title) + ".sql"
		path := filepath.Join(queriesDir, filename)

		var b strings.Builder
		b.WriteString("-- pglet-query\n")
		b.WriteString(fmt.Sprintf("-- @title: %s\n", q.Title))
		if q.Description != "" {
			b.WriteString(fmt.Sprintf("-- @description: %s\n", q.Description))
		}
		db := q.Database
		if db == "" {
			db = "*"
		}
		b.WriteString(fmt.Sprintf("-- @database: %s\n", db))
		if q.Tags != "" {
			b.WriteString(fmt.Sprintf("-- @tags: %s\n", q.Tags))
		}
		b.WriteString("\n")
		b.WriteString(q.SQL)
		if !strings.HasSuffix(q.SQL, "\n") {
			b.WriteString("\n")
		}

		if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
			return fmt.Errorf("write %s: %w", filename, err)
		}
	}
	return nil
}

// ImportSharedQueries reads .pglet/queries/*.sql files and upserts into the repository
func (r *Repository) ImportSharedQueries(dir string) error {
	queriesDir := filepath.Join(dir, "queries")
	entries, err := os.ReadDir(queriesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No queries dir, nothing to import
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(queriesDir, entry.Name()))
		if err != nil {
			continue
		}

		content := string(data)
		if !strings.HasPrefix(content, "-- pglet-query") {
			continue
		}

		title, description, database, tags, sql := parseSharedQueryFile(content)
		if title == "" || sql == "" {
			continue
		}

		if err := r.UpsertSharedQuery(title, sql, description, database, tags); err != nil {
			return fmt.Errorf("import %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func parseSharedQueryFile(content string) (title, description, database, tags, sql string) {
	lines := strings.Split(content, "\n")
	headerDone := false
	var sqlLines []string

	for _, line := range lines {
		if !headerDone {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" && len(sqlLines) == 0 {
				headerDone = true
				continue
			}
			if strings.HasPrefix(trimmed, "-- @title:") {
				title = strings.TrimSpace(strings.TrimPrefix(trimmed, "-- @title:"))
			} else if strings.HasPrefix(trimmed, "-- @description:") {
				description = strings.TrimSpace(strings.TrimPrefix(trimmed, "-- @description:"))
			} else if strings.HasPrefix(trimmed, "-- @database:") {
				database = strings.TrimSpace(strings.TrimPrefix(trimmed, "-- @database:"))
				if database == "*" {
					database = ""
				}
			} else if strings.HasPrefix(trimmed, "-- @tags:") {
				tags = strings.TrimSpace(strings.TrimPrefix(trimmed, "-- @tags:"))
			} else if !strings.HasPrefix(trimmed, "--") {
				headerDone = true
				sqlLines = append(sqlLines, line)
			}
		} else {
			sqlLines = append(sqlLines, line)
		}
	}

	sql = strings.TrimSpace(strings.Join(sqlLines, "\n"))
	return
}

func sanitizeFilename(s string) string {
	// Replace non-alphanumeric chars with underscores
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	result := b.String()
	if result == "" {
		result = "query"
	}
	return result
}
