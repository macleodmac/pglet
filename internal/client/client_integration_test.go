package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
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
			if res.Columns == nil {
				t.Error("columns = nil, want empty slice")
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
		return *res.Rows[0][0]
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

	t.Run("only writes assign a transaction ID", func(t *testing.T) {
		tx, err := c.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err := tx.QueryWithContext(ctx, "SELECT * FROM tx_test"); err != nil {
			t.Fatal(err)
		}
		if wrote, err := tx.HasWrites(ctx); err != nil || wrote {
			t.Errorf("after SELECT: HasWrites = %v, %v; want false", wrote, err)
		}
		if _, err := tx.QueryWithContext(ctx, "DELETE FROM tx_test"); err != nil {
			t.Fatal(err)
		}
		if wrote, err := tx.HasWrites(ctx); err != nil || !wrote {
			t.Errorf("after DELETE: HasWrites = %v, %v; want true", wrote, err)
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

func TestQueryErrorDetail(t *testing.T) {
	c := testClient(t)
	_, err := c.QueryWithContext(context.Background(), "SELECT datnmae FROM pg_database")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HINT") || !strings.Contains(err.Error(), "^") {
		t.Errorf("error lacks hint and position: %v", err)
	}
	if got := SQLState(err); got != "42703" {
		t.Errorf("SQLState = %q, want 42703", got)
	}
}

func TestRelations(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	if _, err := c.QueryWithContext(ctx, `
		DROP SCHEMA IF EXISTS relations_test CASCADE;
		CREATE SCHEMA relations_test;
		CREATE TYPE relations_test.status AS ENUM ('new', 'done');
		CREATE TABLE relations_test.users (id int PRIMARY KEY, email text);
		CREATE TABLE relations_test.tasks (
			id int PRIMARY KEY,
			owner_id int REFERENCES relations_test.users(id),
			status relations_test.status,
			title varchar(80)
		);
		COMMENT ON TABLE relations_test.tasks IS 'Work items';
		COMMENT ON COLUMN relations_test.tasks.title IS 'Short title';
		CREATE VIEW relations_test.open_tasks AS SELECT * FROM relations_test.tasks WHERE status = 'new';
		CREATE TABLE relations_test.events (id int, at date) PARTITION BY RANGE (at);
		CREATE TABLE relations_test.events_2026 PARTITION OF relations_test.events
			FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.QueryWithContext(ctx, "DROP SCHEMA IF EXISTS relations_test CASCADE") })

	rels, err := c.Relations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Relation{}
	for _, r := range rels {
		if r.Schema == "relations_test" {
			byName[r.Name] = r
		}
	}

	if _, ok := byName["events_2026"]; ok {
		t.Error("partition listed alongside its parent")
	}
	if got := byName["open_tasks"].Kind; got != "view" {
		t.Errorf("open_tasks kind = %q, want view", got)
	}

	tasks := byName["tasks"]
	if tasks.Kind != "table" || tasks.Comment != "Work items" {
		t.Errorf("tasks = %q %q, want table with comment", tasks.Kind, tasks.Comment)
	}
	cols := map[string]Column{}
	for _, col := range tasks.Columns {
		cols[col.Name] = col
	}
	if !cols["id"].IsPrimaryKey || cols["owner_id"].IsPrimaryKey {
		t.Error("primary key flags wrong")
	}
	if got := cols["owner_id"].References; got != "relations_test.users.id" {
		t.Errorf("owner_id references %q", got)
	}
	if got := cols["status"].EnumValues; !slices.Equal(got, []string{"new", "done"}) {
		t.Errorf("status enum values = %v", got)
	}
	if got := cols["title"]; got.Type != "character varying(80)" || got.Comment != "Short title" {
		t.Errorf("title = %q %q", got.Type, got.Comment)
	}
}

func ptr(n int64) *int64 { return &n }

func deref(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func TestMixedCaseRelations(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	if _, err := c.QueryWithContext(ctx, `
		DROP SCHEMA IF EXISTS "Mixed Case" CASCADE;
		CREATE SCHEMA "Mixed Case";
		CREATE TABLE "Mixed Case"."Users" (id int);
		CREATE TABLE "Mixed Case".users (id int);
		CREATE TABLE "Mixed Case"."my table" (id int);
		COMMENT ON TABLE "Mixed Case"."Users" IS 'upper';
		COMMENT ON COLUMN "Mixed Case"."Users".id IS 'upper id';
		COMMENT ON COLUMN "Mixed Case".users.id IS 'lower id'`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.QueryWithContext(ctx, `DROP SCHEMA IF EXISTS "Mixed Case" CASCADE`) })

	objects, err := c.Objects(ctx)
	if err != nil {
		t.Fatalf("Objects: %v", err)
	}
	comments := map[string]string{}
	for _, o := range objects["Mixed Case"].Tables {
		comments[o.Name] = o.Comment
	}
	if len(comments) != 3 || comments["Users"] != "upper" || comments["users"] != "" {
		t.Errorf("tables = %v", comments)
	}

	cols, err := c.TableColumns(ctx, "Mixed Case.Users")
	if err != nil {
		t.Fatalf("TableColumns: %v", err)
	}
	if len(cols) != 1 || cols[0].Comment != "upper id" {
		t.Errorf("columns = %+v", cols)
	}
}

func TestQueryValues(t *testing.T) {
	c := testClient(t)
	res, err := c.QueryWithContext(context.Background(), `
		SELECT '\xdeadbeef00ff'::bytea, '2024-01-02 03:04:05.123456'::timestamp,
			'2024-01-02'::date, 9007199254740993::int8, NULL::text`)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, v := range res.Rows[0] {
		if v == nil {
			got = append(got, "NULL")
		} else {
			got = append(got, *v)
		}
	}
	want := []string{`\xdeadbeef00ff`, "2024-01-02 03:04:05.123456", "2024-01-02", "9007199254740993", "NULL"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestQueryTruncatesLargeResults(t *testing.T) {
	c := testClient(t)
	res, err := c.QueryWithContext(context.Background(), fmt.Sprintf("SELECT g FROM generate_series(1, %d) g", MaxRows+5))
	if err != nil {
		t.Fatal(err)
	}
	if res.RowCount != MaxRows || !res.Truncated {
		t.Errorf("row count %d, truncated %v", res.RowCount, res.Truncated)
	}
	if res.RowsAffected == nil || *res.RowsAffected != MaxRows+5 {
		t.Errorf("rows affected = %v, want the full count", res.RowsAffected)
	}
}

func TestSessionStateDoesNotLeak(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	if _, err := c.QueryWithContext(ctx, "SET search_path TO pg_catalog"); err != nil {
		t.Fatal(err)
	}
	for range maxConns {
		res, err := c.QueryWithContext(ctx, "SHOW search_path")
		if err != nil {
			t.Fatal(err)
		}
		if *res.Rows[0][0] == "pg_catalog" {
			t.Fatal("SET from an earlier query leaked into a later one")
		}
	}

	_, err := c.QueryWithContext(ctx, "BEGIN; SELECT 1")
	if !errors.Is(err, ErrLeftTransactionOpen) {
		t.Errorf("BEGIN without COMMIT: err = %v", err)
	}
}
