package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/macleodmac/pglet/internal/client"
)

func testService(t *testing.T) *Service {
	t.Helper()
	url := os.Getenv("PGLET_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("PGLET_TEST_DATABASE_URL not set")
	}
	cl, err := client.New(url)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(nil, "test")
	svc.SetClient(cl)
	t.Cleanup(func() {
		svc.rollbackAllTransactions()
		// The test may have replaced the client, e.g. by reconnecting.
		current := svc.GetClient()
		current.QueryWithContext(context.Background(), "DROP SCHEMA IF EXISTS pglet_test_service CASCADE")
		current.Close()
	})

	if _, err := cl.QueryWithContext(context.Background(), `
		DROP SCHEMA IF EXISTS pglet_test_service CASCADE;
		CREATE SCHEMA pglet_test_service;
		CREATE TABLE pglet_test_service.t (n int);
		INSERT INTO pglet_test_service.t VALUES (1), (2)`); err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestRunQuerySafeWrites(t *testing.T) {
	ctx := context.Background()
	svc := testService(t)
	countRows := func(t *testing.T) int {
		t.Helper()
		res, _, err := svc.RunQuery(ctx, "other-tab", "SELECT * FROM pglet_test_service.t", false)
		if err != nil {
			t.Fatal(err)
		}
		return res.RowCount
	}

	t.Run("reads are not held", func(t *testing.T) {
		_, held, err := svc.RunQuery(ctx, "tab", "SELECT * FROM pglet_test_service.t", true)
		if err != nil {
			t.Fatal(err)
		}
		if held || svc.InTransaction("tab") {
			t.Error("read-only query left a transaction open")
		}
	})

	t.Run("writes are held until rolled back", func(t *testing.T) {
		res, held, err := svc.RunQuery(ctx, "tab", "DELETE FROM pglet_test_service.t WHERE n = 1", true)
		if err != nil {
			t.Fatal(err)
		}
		if !held || !svc.InTransaction("tab") {
			t.Fatal("write was not held for review")
		}
		if res.RowsAffected == nil || *res.RowsAffected != 1 {
			t.Errorf("rows affected = %v, want 1", res.RowsAffected)
		}
		if got := countRows(t); got != 2 {
			t.Errorf("other sessions see %d rows before commit, want 2", got)
		}
		if err := svc.RollbackTransaction("tab"); err != nil {
			t.Fatal(err)
		}
		if got := countRows(t); got != 2 {
			t.Errorf("rows after rollback = %d, want 2", got)
		}
	})

	t.Run("held writes apply on commit", func(t *testing.T) {
		if _, _, err := svc.RunQuery(ctx, "tab", "INSERT INTO pglet_test_service.t VALUES (3)", true); err != nil {
			t.Fatal(err)
		}
		if err := svc.CommitTransaction("tab"); err != nil {
			t.Fatal(err)
		}
		if got := countRows(t); got != 3 {
			t.Errorf("rows after commit = %d, want 3", got)
		}
	})

	t.Run("failed write leaves no transaction", func(t *testing.T) {
		if _, _, err := svc.RunQuery(ctx, "tab", "INSERT INTO pglet_test_service.t VALUES ('x')", true); err == nil {
			t.Fatal("expected error")
		}
		if svc.InTransaction("tab") {
			t.Error("failed query left a transaction open")
		}
	})

	t.Run("without safe writes, writes apply immediately", func(t *testing.T) {
		_, held, err := svc.RunQuery(ctx, "tab", "DELETE FROM pglet_test_service.t WHERE n = 3", false)
		if err != nil {
			t.Fatal(err)
		}
		if held || svc.InTransaction("tab") {
			t.Error("write was held without safe writes")
		}
		if got := countRows(t); got != 2 {
			t.Errorf("rows = %d, want 2", got)
		}
	})
}

func TestWritesCannotEscapeReview(t *testing.T) {
	ctx := context.Background()
	svc := testService(t)
	count := func(t *testing.T) string {
		t.Helper()
		res, _, err := svc.RunQuery(ctx, "counter", "SELECT count(*) FROM pglet_test_service.t", false)
		if err != nil {
			t.Fatal(err)
		}
		return *res.Rows[0][0]
	}

	t.Run("COMMIT inside safe writes is rejected", func(t *testing.T) {
		_, _, err := svc.RunQuery(ctx, "tab", "DELETE FROM pglet_test_service.t; COMMIT", true)
		if !errors.Is(err, ErrTxControl) {
			t.Fatalf("err = %v, want ErrTxControl", err)
		}
		if got := count(t); got != "2" {
			t.Errorf("rows = %s, want 2", got)
		}
	})

	t.Run("COMMIT inside an open transaction is rejected", func(t *testing.T) {
		if err := svc.BeginTransaction(ctx, "tx-tab"); err != nil {
			t.Fatal(err)
		}
		defer svc.RollbackTransaction("tx-tab")
		if _, _, err := svc.RunQuery(ctx, "tx-tab", "COMMIT", false); !errors.Is(err, ErrTxControl) {
			t.Errorf("err = %v, want ErrTxControl", err)
		}
		if !svc.InTransaction("tx-tab") {
			t.Error("transaction was closed")
		}
	})

	t.Run("EXPLAIN ANALYZE is rolled back", func(t *testing.T) {
		if _, err := svc.ExplainQuery(ctx, "tab", "DELETE FROM pglet_test_service.t", true); err != nil {
			t.Fatal(err)
		}
		if got := count(t); got != "2" {
			t.Errorf("rows = %s, want 2", got)
		}
	})

	t.Run("EXPLAIN ANALYZE inside a transaction keeps it usable", func(t *testing.T) {
		if err := svc.BeginTransaction(ctx, "tx-tab"); err != nil {
			t.Fatal(err)
		}
		defer svc.RollbackTransaction("tx-tab")
		if _, _, err := svc.RunQuery(ctx, "tx-tab", "DELETE FROM pglet_test_service.t WHERE n = 1", false); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.ExplainQuery(ctx, "tx-tab", "SELECT * FROM missing_table", true); err == nil {
			t.Fatal("expected error")
		}
		res, _, err := svc.RunQuery(ctx, "tx-tab", "SELECT count(*) FROM pglet_test_service.t", false)
		if err != nil {
			t.Fatalf("transaction aborted by failed explain: %v", err)
		}
		if *res.Rows[0][0] != "1" {
			t.Errorf("rows in transaction = %s, want 1", *res.Rows[0][0])
		}
	})

	t.Run("EXPLAIN refuses a second statement", func(t *testing.T) {
		_, err := svc.ExplainQuery(ctx, "tab", "SELECT 1; DELETE FROM pglet_test_service.t", false)
		if !errors.Is(err, client.ErrMultipleStatements) {
			t.Errorf("err = %v", err)
		}
		if got := count(t); got != "2" {
			t.Errorf("rows = %s, want 2", got)
		}
	})

	t.Run("export is read-only", func(t *testing.T) {
		err := svc.ExportQuery(ctx, "DELETE FROM pglet_test_service.t RETURNING n",
			func(_, _ []string) error { return nil }, func([]*string) error { return nil })
		if client.SQLState(err) != "25006" {
			t.Errorf("err = %v, want read-only transaction error", err)
		}
		if got := count(t); got != "2" {
			t.Errorf("rows = %s, want 2", got)
		}
	})

	t.Run("export streams every row", func(t *testing.T) {
		n := 0
		err := svc.ExportQuery(ctx, fmt.Sprintf("SELECT g FROM generate_series(1, %d) g", client.MaxRows+1),
			func(_, _ []string) error { return nil }, func([]*string) error { n++; return nil })
		if err != nil || n != client.MaxRows+1 {
			t.Errorf("rows = %d, err = %v", n, err)
		}
	})
}

func TestTransactionsDoNotWedgeTheServer(t *testing.T) {
	ctx := context.Background()
	svc := testService(t)

	t.Run("commit still works while a begin waits for a connection", func(t *testing.T) {
		var tabs []string
		for i := range 20 {
			tab := fmt.Sprintf("tab-%d", i)
			if err := svc.BeginTransaction(ctx, tab); err != nil {
				t.Fatalf("begin %s: %v", tab, err)
			}
			tabs = append(tabs, tab)
		}
		waiting := make(chan error, 1)
		go func() { waiting <- svc.BeginTransaction(ctx, "one-too-many") }()

		done := make(chan error, 1)
		go func() { done <- svc.RollbackTransaction(tabs[0]) }()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("rollback blocked behind a waiting begin")
		}
		if err := <-waiting; err != nil {
			t.Errorf("waiting begin: %v", err)
		}
		svc.rollbackAllTransactions()
	})

	t.Run("a second query in the same tab is refused", func(t *testing.T) {
		go svc.RunQuery(ctx, "busy", "SELECT pg_sleep(1)", false)
		time.Sleep(200 * time.Millisecond)
		if _, _, err := svc.RunQuery(ctx, "busy", "SELECT 1", false); !errors.Is(err, ErrQueryRunning) {
			t.Errorf("err = %v, want ErrQueryRunning", err)
		}
		svc.CancelQuery("busy")
	})
}

func TestConnectRollsBackOpenTransactions(t *testing.T) {
	ctx := context.Background()
	svc := testService(t)
	if err := svc.BeginTransaction(ctx, "tab"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Connect(ctx, os.Getenv("PGLET_TEST_DATABASE_URL")); err != nil {
		t.Fatal(err)
	}
	if svc.InTransaction("tab") {
		t.Error("transaction survived reconnecting")
	}
}
