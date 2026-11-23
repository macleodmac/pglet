package client

import (
	"slices"
	"testing"
	"time"
)

func TestSplitStatements(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want [][]string
	}{
		{"single", "select 1", [][]string{{"SELECT", "1"}}},
		{"two", "delete from t; commit", [][]string{{"DELETE", "FROM", "T"}, {"COMMIT"}}},
		{"trailing semicolon and comment", "select 1; -- done\n", [][]string{{"SELECT", "1"}}},
		{"semicolon in string", "select 'a;b'", [][]string{{"SELECT"}}},
		{"doubled quote", "select 'it''s; fine'", [][]string{{"SELECT"}}},
		{"escape string", `select E'\'; commit'`, [][]string{{"SELECT", "E"}}},
		{"quoted identifier", `select 1 as "x;y"`, [][]string{{"SELECT", "1", "AS"}}},
		{"dollar quoted", "do $$ begin; commit; end $$", [][]string{{"DO"}}},
		{"tagged dollar quote", "create function f() returns int as $fn$ select 1; $fn$ language sql", [][]string{{"CREATE", "FUNCTION", "F"}}},
		{"positional parameter", "prepare p as select $1; execute p(1)", [][]string{{"PREPARE", "P", "AS"}, {"EXECUTE", "P"}}},
		{"nested block comment", "/* a /* b; */ c; */ select 1", [][]string{{"SELECT", "1"}}},
		{"leading comment", "-- note\nbegin", [][]string{{"BEGIN"}}},
		{"only comments", "-- nothing; here\n/* ; */", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got [][]string
			for _, st := range SplitStatements(tc.sql) {
				got = append(got, st.Keywords)
			}
			if !slices.EqualFunc(got, tc.want, slices.Equal) {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsTransactionControl(t *testing.T) {
	for sql, want := range map[string]bool{
		"BEGIN":                     true,
		"start transaction":         true,
		"commit and chain":          true,
		"end":                       true,
		"abort":                     true,
		"rollback":                  true,
		"rollback to savepoint s":   false,
		"savepoint s":               false,
		"release savepoint s":       false,
		"prepare transaction 'x'":   true,
		"prepare p as select 1":     false,
		"select 'commit'":           false,
		"/* commit */ select 1":     false,
		"update t set ended = true": false,
	} {
		st, err := SingleStatement(sql)
		if err != nil {
			t.Fatalf("%q: %v", sql, err)
		}
		if got := st.IsTransactionControl(); got != want {
			t.Errorf("%q: got %v, want %v", sql, got, want)
		}
	}
}

func TestSingleStatement(t *testing.T) {
	if _, err := SingleStatement("select 1; delete from t"); err != ErrMultipleStatements {
		t.Errorf("two statements: err = %v", err)
	}
	if _, err := SingleStatement("  -- x\n"); err != ErrEmptyQuery {
		t.Errorf("empty: err = %v", err)
	}
	st, err := SingleStatement("select 1;\n")
	if err != nil || st.SQL != "select 1" {
		t.Errorf("got %q, %v", st.SQL, err)
	}
}

func TestFormatValue(t *testing.T) {
	ist := time.FixedZone("", 5*3600+30*60)
	ts := time.Date(2024, 1, 2, 3, 4, 5, 123456000, time.UTC)
	cases := []struct {
		v    any
		typ  string
		want string
	}{
		{[]byte{0xde, 0xad, 0x00, 0xff}, "BYTEA", `\xdead00ff`},
		{[]byte("12.5000"), "NUMERIC", "12.5000"},
		{int64(9007199254740993), "INT8", "9007199254740993"},
		{true, "BOOL", "true"},
		{1e6, "FLOAT8", "1000000"},
		{0.1, "FLOAT4", "0.1"},
		{1e-7, "FLOAT8", "1e-07"},
		{ts, "TIMESTAMP", "2024-01-02 03:04:05.123456"},
		{ts, "TIMESTAMPTZ", "2024-01-02 03:04:05.123456+00"},
		{ts.In(ist), "TIMESTAMPTZ", "2024-01-02 08:34:05.123456+05:30"},
		{time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), "DATE", "2024-01-02"},
		{time.Date(0, 1, 1, 13, 0, 0, 0, time.UTC), "TIME", "13:00:00"},
	}
	for _, tc := range cases {
		if got := formatValue(tc.v, tc.typ); got != tc.want {
			t.Errorf("formatValue(%v, %s) = %q, want %q", tc.v, tc.typ, got, tc.want)
		}
	}
}

func TestRedactURL(t *testing.T) {
	got := RedactURL(`ping: parse "postgres://pglet:S3cret%zz@localhost:5432/demo": invalid URL escape`)
	want := `ping: parse "postgres://pglet:xxxxx@localhost:5432/demo": invalid URL escape`
	if got != want {
		t.Errorf("got %q", got)
	}
	if got := RedactURL("postgres://pglet@localhost/demo"); got != "postgres://pglet@localhost/demo" {
		t.Errorf("user-only URL changed: %q", got)
	}
}
