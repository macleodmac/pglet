package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// setupTable creates a fresh table and drops it when the test ends.
func setupTable(t *testing.T, c *Client, name, ddl string) {
	t.Helper()
	ctx := context.Background()
	for _, q := range []string{"DROP TABLE IF EXISTS " + name, ddl} {
		if _, err := c.QueryWithContext(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	t.Cleanup(func() { c.QueryWithContext(ctx, "DROP TABLE IF EXISTS "+name) })
}

func scalar(t *testing.T, c *Client, q string) string {
	t.Helper()
	res, err := c.QueryWithContext(context.Background(), q)
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return fmt.Sprint(res.Rows[0][0])
}

func TestQueryLimited(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	setupTable(t, c, "limit_test", "CREATE TABLE limit_test (id int PRIMARY KEY)")

	t.Run("select over the cap is truncated with the full count", func(t *testing.T) {
		res, err := c.QueryLimited(ctx, "SELECT g FROM generate_series(1, 2500) g", 1000)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Truncated || res.RowCount != 1000 || len(res.Rows) != 1000 {
			t.Fatalf("truncated=%v rows=%d, want truncated with 1000 rows", res.Truncated, res.RowCount)
		}
		if res.RowsAffected == nil || *res.RowsAffected != 2500 {
			t.Errorf("affected = %v, want 2500", deref(res.RowsAffected))
		}
		if res.Rows[999][0] != int64(1000) {
			t.Errorf("last kept row = %v, want 1000", res.Rows[999][0])
		}
	})

	t.Run("exactly at the cap is not truncated", func(t *testing.T) {
		res, err := c.QueryLimited(ctx, "SELECT g FROM generate_series(1, 10) g", 10)
		if err != nil {
			t.Fatal(err)
		}
		if res.Truncated || res.RowCount != 10 {
			t.Errorf("truncated=%v rows=%d, want 10 untruncated", res.Truncated, res.RowCount)
		}
	})

	t.Run("zero means no cap", func(t *testing.T) {
		res, err := c.QueryLimited(ctx, "SELECT g FROM generate_series(1, 2500) g", 0)
		if err != nil {
			t.Fatal(err)
		}
		if res.Truncated || res.RowCount != 2500 {
			t.Errorf("truncated=%v rows=%d, want 2500", res.Truncated, res.RowCount)
		}
	})

	t.Run("truncated RETURNING still applies every row", func(t *testing.T) {
		res, err := c.QueryLimited(ctx, "INSERT INTO limit_test SELECT generate_series(1, 50) RETURNING id", 5)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Truncated || res.RowCount != 5 {
			t.Errorf("truncated=%v rows=%d, want 5 truncated", res.Truncated, res.RowCount)
		}
		if res.CommandTag != "INSERT" || res.RowsAffected == nil || *res.RowsAffected != 50 {
			t.Errorf("tag=%q affected=%v, want INSERT 50", res.CommandTag, deref(res.RowsAffected))
		}
		if got := scalar(t, c, "SELECT count(*) FROM limit_test"); got != "50" {
			t.Errorf("count = %s, want 50", got)
		}
	})

	t.Run("cancel while draining returns the kept rows", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		// Rows past the cap arrive slowly, so the drain is still running at the
		// deadline. They are wide so the server flushes each one rather than
		// buffering them.
		res, err := c.QueryLimited(ctx,
			"SELECT g, repeat('x', 20000), CASE WHEN g > 3 THEN pg_sleep(0.1) END FROM generate_series(1, 100) g", 3)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Truncated || res.RowCount != 3 || res.RowsAffected != nil {
			t.Errorf("truncated=%v rows=%d affected=%v, want 3 rows, unknown total",
				res.Truncated, res.RowCount, deref(res.RowsAffected))
		}
		if _, err := c.QueryWithContext(context.Background(), "SELECT 1"); err != nil {
			t.Fatalf("pool unusable after cancel: %v", err)
		}
	})

	t.Run("cap applies inside a transaction", func(t *testing.T) {
		tx, err := c.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		res, err := tx.QueryLimited(ctx, "SELECT g FROM generate_series(1, 20) g", 7)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Truncated || res.RowCount != 7 || deref(res.RowsAffected) != int64(20) {
			t.Errorf("truncated=%v rows=%d affected=%v", res.Truncated, res.RowCount, deref(res.RowsAffected))
		}
		if _, err := tx.QueryWithContext(ctx, "SELECT 1"); err != nil {
			t.Errorf("transaction unusable after truncated read: %v", err)
		}
	})
}

func TestQueryRolledBack(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	setupTable(t, c, "rb_test", "CREATE TABLE rb_test (id int PRIMARY KEY)")
	if _, err := c.QueryWithContext(ctx, "INSERT INTO rb_test SELECT generate_series(1, 10)"); err != nil {
		t.Fatal(err)
	}

	t.Run("explain analyze of a delete keeps the rows", func(t *testing.T) {
		res, err := c.QueryRolledBack(ctx, "EXPLAIN (ANALYZE, FORMAT JSON) DELETE FROM rb_test")
		if err != nil {
			t.Fatal(err)
		}
		if res.RowCount != 1 || !strings.Contains(fmt.Sprint(res.Rows[0][0]), `"Actual Rows"`) {
			t.Errorf("unexpected plan: %v", res.Rows)
		}
		if got := scalar(t, c, "SELECT count(*) FROM rb_test"); got != "10" {
			t.Errorf("count = %s, want 10", got)
		}
	})

	t.Run("multiple statements are rejected", func(t *testing.T) {
		_, err := c.QueryRolledBack(ctx, "EXPLAIN SELECT 1; DELETE FROM rb_test")
		if err == nil || !strings.Contains(err.Error(), "multiple commands") {
			t.Fatalf("err = %v, want multiple commands error", err)
		}
		if got := scalar(t, c, "SELECT count(*) FROM rb_test"); got != "10" {
			t.Errorf("count = %s, want 10", got)
		}
	})

	t.Run("trailing semicolon is fine", func(t *testing.T) {
		if _, err := c.QueryRolledBack(ctx, "EXPLAIN (FORMAT JSON) SELECT 1;"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("inside a transaction: sees its work, undoes the statement, survives errors", func(t *testing.T) {
		tx, err := c.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err := tx.QueryWithContext(ctx, "INSERT INTO rb_test VALUES (100)"); err != nil {
			t.Fatal(err)
		}
		res, err := tx.QueryRolledBack(ctx, "EXPLAIN (ANALYZE, FORMAT JSON) DELETE FROM rb_test")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(fmt.Sprint(res.Rows[0][0]), `"Actual Rows": 11`) {
			t.Errorf("plan should see the uncommitted row: %v", res.Rows[0][0])
		}
		if _, err := tx.QueryRolledBack(ctx, "EXPLAIN SELECT * FROM no_such_table"); err == nil {
			t.Fatal("expected error")
		}
		res, err = tx.QueryWithContext(ctx, "SELECT count(*) FROM rb_test")
		if err != nil {
			t.Fatalf("transaction aborted by explain error: %v", err)
		}
		if got := fmt.Sprint(res.Rows[0][0]); got != "11" {
			t.Errorf("count in tx = %s, want 11", got)
		}
	})
}

func TestOptions(t *testing.T) {
	url := os.Getenv("PGLET_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("PGLET_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	rw := testClient(t)
	setupTable(t, rw, "ro_test", "CREATE TABLE ro_test (id int PRIMARY KEY, v text)")

	ro, err := rw.WithOptions(Options{ReadOnly: true, StatementTimeoutMs: 100})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ro.Close)

	if _, err := ro.QueryWithContext(ctx, "INSERT INTO ro_test VALUES (1, 'a')"); err == nil ||
		!strings.Contains(err.Error(), "read-only") {
		t.Errorf("insert err = %v, want read-only error", err)
	}
	if _, err := ro.QueryWithContext(ctx, "SELECT pg_sleep(1)"); err == nil ||
		!strings.Contains(err.Error(), "statement timeout") {
		t.Errorf("sleep err = %v, want statement timeout", err)
	}
	if got := scalar(t, ro, "SELECT count(*) FROM ro_test"); got != "0" {
		t.Errorf("reads should work, got %s", got)
	}
	if info, _ := ro.Info(); !info.Options.ReadOnly || info.Options.StatementTimeoutMs != 100 {
		t.Errorf("info options = %+v", info.Options)
	}
	// Transactions opened from the read-only pool are read-only too.
	tx, err := ro.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.QueryWithContext(ctx, "DELETE FROM ro_test"); err == nil {
		t.Error("delete in read-only transaction succeeded")
	}
	tx.Rollback()

	rw.QueryWithContext(ctx, "INSERT INTO ro_test VALUES (1, 'a')")
	v := "b"
	if err := ro.UpdateRow(ctx, "ro_test", map[string]*string{"id": ptrStr("1")}, map[string]*string{"v": &v}); err == nil ||
		!strings.Contains(err.Error(), "read-only") {
		t.Errorf("UpdateRow err = %v, want read-only error", err)
	}
}

func ptrStr(s string) *string { return &s }

func TestWithRuntimeParams(t *testing.T) {
	tests := []struct {
		in   string
		opts Options
		want string
	}{
		{"postgres://u@h/db", Options{}, "postgres://u@h/db"},
		{"postgres://u@h/db?sslmode=disable", Options{ReadOnly: true},
			"postgres://u@h/db?default_transaction_read_only=on&sslmode=disable"},
		{"postgres://u@h/db", Options{StatementTimeoutMs: 5000}, "postgres://u@h/db?statement_timeout=5000"},
		{"host=h dbname=db", Options{ReadOnly: true}, "host=h dbname=db default_transaction_read_only=on"},
	}
	for _, tt := range tests {
		if got := withRuntimeParams(tt.in, tt.opts); got != tt.want {
			t.Errorf("withRuntimeParams(%q, %+v) = %q, want %q", tt.in, tt.opts, got, tt.want)
		}
	}
}

func TestTableRowsFilters(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	setupTable(t, c, "filter_test", "CREATE TABLE filter_test (id int PRIMARY KEY, name text, born date)")
	if _, err := c.QueryWithContext(ctx, `INSERT INTO filter_test VALUES
		(1, 'Ada', '1815-12-10'), (2, 'Alan', '1912-06-23'), (3, 'Grace', NULL), (4, 'ada', '2000-01-01')`); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		filters []Filter
		want    int
	}{
		{"none", nil, 4},
		{"eq int", []Filter{{"id", OpEq, "2"}}, 1},
		{"gt date", []Filter{{"born", OpGt, "1900-01-01"}}, 2},
		{"like is case sensitive", []Filter{{"name", OpLike, "A%"}}, 2},
		{"ilike", []Filter{{"name", OpILike, "ada"}}, 2},
		{"like on int casts", []Filter{{"id", OpLike, "1%"}}, 1},
		{"is null", []Filter{{"born", OpIsNull, ""}}, 1},
		{"combined with AND", []Filter{{"name", OpILike, "a%"}, {"born", OpNotNull, ""}, {"id", OpNeq, "4"}}, 2},
		{"injection attempt is just a value", []Filter{{"name", OpEq, "x' OR '1'='1"}}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, total, err := c.TableRows(ctx, "filter_test", 100, 0, "id", "ASC", tt.filters)
			if err != nil {
				t.Fatal(err)
			}
			if total != tt.want || res.RowCount != tt.want {
				t.Errorf("total=%d rows=%d, want %d", total, res.RowCount, tt.want)
			}
		})
	}

	t.Run("bad value for column type errors", func(t *testing.T) {
		if _, _, err := c.TableRows(ctx, "filter_test", 100, 0, "", "", []Filter{{"id", OpEq, "abc"}}); err == nil {
			t.Error("expected error")
		}
	})
	t.Run("unknown operator errors", func(t *testing.T) {
		if _, _, err := c.TableRows(ctx, "filter_test", 100, 0, "", "", []Filter{{"id", "drop", "1"}}); err == nil {
			t.Error("expected error")
		}
	})
}

func TestUpdateRow(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	setupTable(t, c, "edit_test", "CREATE TABLE edit_test (a int, b text, v text, n int, PRIMARY KEY (a, b))")
	setupTable(t, c, "nopk_test", "CREATE TABLE nopk_test (v text)")
	if _, err := c.QueryWithContext(ctx, "INSERT INTO edit_test VALUES (1, 'x', 'old', 5), (1, 'y', 'keep', 6)"); err != nil {
		t.Fatal(err)
	}
	s := func(v string) *string { return &v }

	if err := c.UpdateRow(ctx, "edit_test",
		map[string]*string{"a": s("1"), "b": s("x")},
		map[string]*string{"v": s("new"), "n": nil}); err != nil {
		t.Fatal(err)
	}
	if got := scalar(t, c, "SELECT v || ':' || coalesce(n::text, 'NULL') FROM edit_test WHERE b = 'x'"); got != "new:NULL" {
		t.Errorf("row = %s, want new:NULL", got)
	}
	if got := scalar(t, c, "SELECT v FROM edit_test WHERE b = 'y'"); got != "keep" {
		t.Errorf("other row changed to %s", got)
	}

	errorCases := []struct {
		name    string
		table   string
		key     map[string]*string
		changes map[string]*string
		want    string
	}{
		{"partial key", "edit_test", map[string]*string{"a": s("1")}, map[string]*string{"v": s("z")}, "primary key"},
		{"non-key column as key", "edit_test", map[string]*string{"a": s("1"), "v": s("keep")}, map[string]*string{"v": s("z")}, "missing primary key column"},
		{"no match", "edit_test", map[string]*string{"a": s("9"), "b": s("x")}, map[string]*string{"v": s("z")}, "exactly one row"},
		{"no changes", "edit_test", map[string]*string{"a": s("1"), "b": s("x")}, nil, "no changes"},
		{"no primary key", "nopk_test", map[string]*string{"v": s("a")}, map[string]*string{"v": s("z")}, "no primary key"},
		{"bad value", "edit_test", map[string]*string{"a": s("1"), "b": s("x")}, map[string]*string{"n": s("abc")}, "invalid input"},
	}
	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			err := c.UpdateRow(ctx, tt.table, tt.key, tt.changes)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
	if got := scalar(t, c, "SELECT string_agg(v, ',' ORDER BY b) FROM edit_test"); got != "new,keep" {
		t.Errorf("failed updates changed data: %s", got)
	}
	if !errors.Is(c.UpdateRow(ctx, "edit_test", map[string]*string{"a": s("9"), "b": s("x")}, map[string]*string{"v": s("z")}), ErrRowNotUnique) {
		t.Error("no-match error should wrap ErrRowNotUnique")
	}
}
