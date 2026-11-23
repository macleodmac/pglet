package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSharedQueries(t *testing.T) {
	dir := t.TempDir()
	shared := filepath.Join(dir, "queries")
	if err := os.MkdirAll(shared, 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(shared, "active.sql")
	os.WriteFile(file, []byte("-- pglet-query\n-- @title: Active users\n-- @database: *\n\nSELECT 1;\n"), 0644)

	open := func(t *testing.T) *Repository {
		t.Helper()
		r, err := Open(filepath.Join(dir, "pglet.db"), shared)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { r.Close() })
		return r
	}

	r := open(t)
	list, err := r.ListSavedQueries("")
	if err != nil || len(list) != 1 || list[0].ID != "shared:active.sql" || !list[0].Shared {
		t.Fatalf("list = %+v, %v", list, err)
	}

	t.Run("edits are written to the file and survive a restart", func(t *testing.T) {
		q := list[0]
		q.SQL = "SELECT 2;"
		q.Description = "line one\nline two"
		if err := r.UpdateSavedQuery(q); err != nil {
			t.Fatal(err)
		}
		r.Close()
		r = open(t)
		got, err := r.GetSavedQuery(q.ID)
		if err != nil || got.SQL != "SELECT 2;" || got.Description != "line one line two" {
			t.Errorf("got %+v, %v", got, err)
		}
	})

	t.Run("paths outside the shared directory are refused", func(t *testing.T) {
		for _, id := range []string{"shared:../pglet.db", "shared:../../etc/passwd.sql", "shared:sub/x.sql", "shared:"} {
			if _, err := r.GetSavedQuery(id); !errors.Is(err, ErrNotFound) {
				t.Errorf("%s: err = %v", id, err)
			}
		}
	})

	t.Run("delete removes the file", func(t *testing.T) {
		if err := r.DeleteSavedQuery("shared:active.sql"); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(file); !os.IsNotExist(err) {
			t.Errorf("file still exists: %v", err)
		}
	})
}

func TestOpenFailsFastWhenLocked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pglet.db")
	r, err := Open(path, "")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err := Open(path, ""); err == nil {
		t.Fatal("second Open succeeded")
	}
}
