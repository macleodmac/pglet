package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

func testClient(t *testing.T) *Client {
	t.Helper()
	url := os.Getenv("PGLET_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("PGLET_TEST_DATABASE_URL not set")
	}
	c, err := New(url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestQueryCommandTags(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()

	mustQuery := func(t *testing.T, q string) *QueryResult {
		t.Helper()
		res, err := c.QueryWithContext(ctx, q)
		if err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return res
	}
	mustQuery(t, "DROP TABLE IF EXISTS tag_test")
	mustQuery(t, "CREATE TABLE tag_test (id int PRIMARY KEY, v text)")
	t.Cleanup(func() { c.QueryWithContext(ctx, "DROP TABLE IF EXISTS tag_test") })

	tests := []struct {
		query    string
		tag      string
		affected *int64
		rows     int
	}{
		{"INSERT INTO tag_test VALUES (1,'a'),(2,'b'),(3,'c')", "INSERT", ptr(3), 0},
		{"UPDATE tag_test SET v = 'x' WHERE id < 3", "UPDATE", ptr(2), 0},
		{"UPDATE tag_test SET v = 'y' WHERE id = 3 RETURNING id", "UPDATE", ptr(1), 1},
		{"SELECT * FROM tag_test", "SELECT", ptr(3), 3},
		{"MERGE INTO tag_test t USING (VALUES (1)) s(id) ON t.id = s.id WHEN MATCHED THEN UPDATE SET v = 'm'", "MERGE", ptr(1), 0},
		{"DELETE FROM tag_test WHERE id = 1", "DELETE", ptr(1), 0},
		{"CREATE INDEX tag_test_v ON tag_test (v)", "CREATE INDEX", nil, 0},
		{"UPDATE tag_test SET v = 'z'; DELETE FROM tag_test", "DELETE", ptr(2), 0},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			res := mustQuery(t, tt.query)
			if res.CommandTag != tt.tag {
				t.Errorf("tag = %q, want %q", res.CommandTag, tt.tag)
			}
			if (res.RowsAffected == nil) != (tt.affected == nil) ||
				(tt.affected != nil && *res.RowsAffected != *tt.affected) {
				t.Errorf("affected = %v, want %v", deref(res.RowsAffected), deref(tt.affected))
			}
			if res.RowCount != tt.rows {
				t.Errorf("rows = %d, want %d", res.RowCount, tt.rows)
			}
		})
	}

	t.Run("error in later statement of a batch is reported", func(t *testing.T) {
		if _, err := c.QueryWithContext(ctx, "SELECT 1; SELECT 1/0"); err == nil {
			t.Fatal("expected division by zero error")
		}
	})
}

func TestQueryCancel(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	if _, err := c.QueryWithContext(ctx, "SELECT pg_sleep(10)"); err == nil {
		t.Fatal("expected cancellation error")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("query ran for %s after cancel", elapsed)
	}
	if _, err := c.QueryWithContext(context.Background(), "SELECT 1"); err != nil {
		t.Fatalf("pool unusable after cancel: %v", err)
	}
}

func TestTransaction(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()

	if _, err := c.QueryWithContext(ctx, "CREATE TABLE IF NOT EXISTS tx_test (id int)"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.QueryWithContext(ctx, "DROP TABLE IF EXISTS tx_test") })

	count := func(t *testing.T) string {
		t.Helper()
		res, err := c.QueryWithContext(ctx, "SELECT count(*) FROM tx_test")
		if err != nil {
			t.Fatal(err)
		}
		return fmt.Sprint(res.Rows[0][0])
	}

	t.Run("rollback discards changes", func(t *testing.T) {
		tx, err := c.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		res, err := tx.QueryWithContext(ctx, "INSERT INTO tx_test VALUES (1),(2)")
		if err != nil {
			t.Fatal(err)
		}
		if res.RowsAffected == nil || *res.RowsAffected != 2 {
			t.Errorf("affected = %v, want 2", deref(res.RowsAffected))
		}
		if err := tx.Rollback(); err != nil {
			t.Fatal(err)
		}
		if got := count(t); got != "0" {
			t.Errorf("count after rollback = %s, want 0", got)
		}
	})

	t.Run("commit persists changes", func(t *testing.T) {
		tx, err := c.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.QueryWithContext(ctx, "INSERT INTO tx_test VALUES (1)"); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		if got := count(t); got != "1" {
			t.Errorf("count after commit = %s, want 1", got)
		}
	})

	t.Run("commit of aborted transaction reports rollback", func(t *testing.T) {
		tx, err := c.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.QueryWithContext(ctx, "INSERT INTO tx_test VALUES (99)"); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.QueryWithContext(ctx, "SELECT 1/0"); err == nil {
			t.Fatal("expected error")
		}
		if err := tx.Commit(); !errors.Is(err, ErrTxAborted) {
			t.Fatalf("commit err = %v, want ErrTxAborted", err)
		}
		if got := count(t); got != "1" {
			t.Errorf("count = %s, want 1", got)
		}
	})
}

func ptr(n int64) *int64 { return &n }

func deref(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}
