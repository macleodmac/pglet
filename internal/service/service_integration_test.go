package service

import (
	"context"
	"os"
	"testing"

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
		cl.QueryWithContext(context.Background(), "DROP SCHEMA IF EXISTS pglet_test_service CASCADE")
		cl.Close()
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
