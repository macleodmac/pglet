package client

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

type Client struct {
	db      *sql.DB
	connURL string
	opts    Options
}

// Options are session settings applied to every pooled connection.
type Options struct {
	// ReadOnly sets default_transaction_read_only, so writes fail unless a
	// session explicitly opts back in. It is a guard rail, not a permission.
	ReadOnly bool
	// StatementTimeoutMs sets statement_timeout; 0 leaves the server default.
	StatementTimeoutMs int
}

type ConnectionInfo struct {
	Host     string
	Port     int
	User     string
	Database string
	Version  string
	Options  Options
}

type QueryResult struct {
	Columns     []string
	ColumnTypes []string
	Rows        [][]any
	RowCount    int
	// Truncated is set when the result had more rows than the requested cap.
	Truncated  bool
	DurationMs int64
	// CommandTag is the verb of the final command, e.g. "UPDATE".
	CommandTag string
	// RowsAffected is nil for commands that do not report a count (DDL, BEGIN, ...).
	// For a truncated SELECT it is the full row count.
	RowsAffected *int64

	cancelled bool
}

func New(connURL string, opts Options) (*Client, error) {
	db, err := sql.Open("postgres", withRuntimeParams(connURL, opts))
	if err != nil {
		return nil, fmt.Errorf("open connection: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	return &Client{db: db, connURL: connURL, opts: opts}, nil
}

// withRuntimeParams adds the options as run-time parameters, which lib/pq
// passes through to the server in the startup message.
func withRuntimeParams(connURL string, opts Options) string {
	params := map[string]string{}
	if opts.ReadOnly {
		params["default_transaction_read_only"] = "on"
	}
	if opts.StatementTimeoutMs > 0 {
		params["statement_timeout"] = strconv.Itoa(opts.StatementTimeoutMs)
	}
	if len(params) == 0 {
		return connURL
	}

	if u, err := url.Parse(connURL); err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		return u.String()
	}
	// key=value DSN: later keys win.
	for k, v := range params {
		connURL += fmt.Sprintf(" %s=%s", k, v)
	}
	return connURL
}

func (c *Client) Options() Options {
	return c.opts
}

func (c *Client) Close() {
	if c.db != nil {
		c.db.Close()
	}
}

func (c *Client) Database() string {
	u, err := url.Parse(c.connURL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.Path, "/")
}

func (c *Client) Info() (*ConnectionInfo, error) {
	info := &ConnectionInfo{Options: c.opts}

	u, err := url.Parse(c.connURL)
	if err == nil {
		info.Host = u.Hostname()
		fmt.Sscanf(u.Port(), "%d", &info.Port)
		if u.User != nil {
			info.User = u.User.Username()
		}
		info.Database = strings.TrimPrefix(u.Path, "/")
	}

	var version string
	if err := c.db.QueryRow("SHOW server_version").Scan(&version); err == nil {
		info.Version = version
	}

	return info, nil
}

func (c *Client) SwitchDatabase(database string) (*Client, error) {
	u, err := url.Parse(c.connURL)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	u.Path = "/" + database
	return New(u.String(), c.opts)
}

// WithOptions opens a new client on the same database with different options.
func (c *Client) WithOptions(opts Options) (*Client, error) {
	return New(c.connURL, opts)
}

func (c *Client) Databases() ([]string, error) {
	rows, err := c.db.Query("SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dbs []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		dbs = append(dbs, name)
	}
	return dbs, rows.Err()
}

func (c *Client) QueryWithContext(ctx context.Context, query string) (*QueryResult, error) {
	return c.queryContext(ctx, query, queryOpts{})
}

// QueryLimited is QueryWithContext with at most maxRows rows kept; see readRows.
func (c *Client) QueryLimited(ctx context.Context, query string, maxRows int) (*QueryResult, error) {
	return c.queryContext(ctx, query, queryOpts{maxRows: maxRows})
}

// QueryRolledBack runs a single statement in a transaction that is always
// rolled back, so nothing it does is kept. Used for EXPLAIN ANALYZE.
func (c *Client) QueryRolledBack(ctx context.Context, query string) (*QueryResult, error) {
	tx, err := c.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return runQuery(ctx, tx.conn, query, queryOpts{singleStatement: true})
}

// queryContext executes a query on a connection borrowed from the shared pool.
func (c *Client) queryContext(ctx context.Context, query string, opts queryOpts) (*QueryResult, error) {
	conn, err := c.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return runQuery(ctx, conn, query, opts)
}

type queryOpts struct {
	// maxRows caps the rows kept in memory; 0 keeps them all.
	maxRows int
	// singleStatement uses the extended protocol, where the server rejects
	// a string holding more than one statement.
	singleStatement bool
	// args are bound parameters ($1, $2, ...). Any args imply the extended
	// protocol, so a single statement.
	args []any
}

// runQuery executes a query and returns columns, types, rows and the command
// tag. It talks to the driver directly because database/sql hides the tag.
func runQuery(ctx context.Context, conn *sql.Conn, query string, opts queryOpts) (*QueryResult, error) {
	start := time.Now()
	var result *QueryResult
	err := conn.Raw(func(dc any) error {
		rows, closeStmt, err := driverQuery(ctx, dc, query, opts)
		if err != nil {
			return err
		}
		defer closeStmt()
		result, err = readRows(ctx, rows, opts.maxRows)
		if cerr := rows.Close(); err == nil && !(result != nil && result.cancelled) {
			err = cerr
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func driverQuery(ctx context.Context, dc any, query string, opts queryOpts) (driver.Rows, func(), error) {
	noop := func() {}
	var args []driver.NamedValue
	for i, a := range opts.args {
		args = append(args, driver.NamedValue{Ordinal: i + 1, Value: a})
	}

	// lib/pq already uses the extended protocol when there are args.
	if !opts.singleStatement || len(args) > 0 {
		queryer, ok := dc.(driver.QueryerContext)
		if !ok {
			return nil, noop, errors.New("driver does not support QueryerContext")
		}
		rows, err := queryer.QueryContext(ctx, query, args)
		return rows, noop, err
	}

	preparer, ok := dc.(driver.ConnPrepareContext)
	if !ok {
		return nil, noop, errors.New("driver does not support ConnPrepareContext")
	}
	stmt, err := preparer.PrepareContext(ctx, query)
	if err != nil {
		return nil, noop, err
	}
	sq, ok := stmt.(driver.StmtQueryContext)
	if !ok {
		stmt.Close()
		return nil, noop, errors.New("driver does not support StmtQueryContext")
	}
	rows, err := sq.QueryContext(ctx, nil)
	if err != nil {
		stmt.Close()
		return nil, noop, err
	}
	return rows, func() { stmt.Close() }, nil
}

// readRows reads the first result set. The command tag is captured before the
// rows are closed so it describes that result set, not a later statement.
//
// With maxRows > 0, rows past the cap are read and counted but not kept: that
// bounds memory without cutting a statement short, which would abort an
// UPDATE ... RETURNING half way. If ctx is cancelled while draining, the rows
// already kept are returned rather than an error.
func readRows(ctx context.Context, rows driver.Rows, maxRows int) (*QueryResult, error) {
	columns := rows.Columns()
	typeNames := make([]string, len(columns))
	if typed, ok := rows.(driver.RowsColumnTypeDatabaseTypeName); ok {
		for i := range columns {
			typeNames[i] = typed.ColumnTypeDatabaseTypeName(i)
		}
	}

	var data [][]any
	truncated, cancelled := false, false
	discard := make([]driver.Value, len(columns))
	for {
		vals := discard
		if !truncated {
			vals = make([]driver.Value, len(columns))
		}
		if err := rows.Next(vals); err == io.EOF {
			break
		} else if err != nil {
			if truncated && ctx.Err() != nil {
				cancelled = true
				break
			}
			return nil, err
		}
		if truncated {
			continue
		}
		if maxRows > 0 && len(data) == maxRows {
			truncated = true
			continue
		}
		row := make([]any, len(vals))
		for i, v := range vals {
			// The driver reuses its read buffer, so bytes must be copied out.
			if b, ok := v.([]byte); ok {
				row[i] = string(b)
			} else {
				row[i] = v
			}
		}
		data = append(data, row)
	}

	result := &QueryResult{
		Columns:     columns,
		ColumnTypes: typeNames,
		Rows:        data,
		RowCount:    len(data),
		Truncated:   truncated,
		cancelled:   cancelled,
	}
	if cancelled {
		// The total is unknown and the tag never arrived.
		return result, nil
	}
	if tagged, ok := rows.(interface {
		Tag() string
		Result() driver.Result
	}); ok {
		result.CommandTag, result.RowsAffected = parseCommandTag(tagged.Tag(), tagged.Result())
	}
	return result, nil
}

// countedCommands are the tags lib/pq strips the row count from and exposes
// via driver.Result instead.
var countedCommands = map[string]bool{
	"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true,
	"COPY": true, "FETCH": true, "MOVE": true,
}

func parseCommandTag(tag string, res driver.Result) (string, *int64) {
	if countedCommands[tag] {
		n, err := res.RowsAffected()
		if err != nil {
			return tag, nil
		}
		return tag, &n
	}
	// Commands lib/pq does not know about (e.g. MERGE) keep the count in the tag.
	if verb, count, ok := strings.Cut(tag, " "); ok {
		if n, err := strconv.ParseInt(count, 10, 64); err == nil {
			return verb, &n
		}
	}
	return tag, nil
}

// splitTableName splits "schema.table" into schema and table parts.
func splitTableName(table string) (string, string) {
	parts := strings.SplitN(table, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "public", parts[0]
}

// quoteIdent quotes a PostgreSQL identifier.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
