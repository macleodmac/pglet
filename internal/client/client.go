package client

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

type Client struct {
	db      *sql.DB
	connURL string
}

// MaxRows caps how many rows a query result holds; the rest are discarded and
// the result is marked truncated.
const MaxRows = 10_000

// maxConns bounds the pool. Each open transaction pins one connection.
const maxConns = 20

// acquireTimeout bounds the wait for a free pooled connection.
const acquireTimeout = 10 * time.Second

// ErrPoolExhausted means every pooled connection is in use, typically because
// each is pinned by an open transaction.
var ErrPoolExhausted = fmt.Errorf("all %d database connections are in use, mostly by open transactions; commit or roll back one and try again", maxConns)

type ConnectionInfo struct {
	Host     string
	Port     int
	User     string
	Database string
	Version  string
}

type QueryResult struct {
	Columns     []string
	ColumnTypes []string
	Rows        [][]*string
	RowCount    int
	DurationMs  int64
	// CommandTag is the verb of the final command, e.g. "UPDATE".
	CommandTag string
	// RowsAffected is nil for commands that do not report a count (DDL, BEGIN, ...).
	RowsAffected *int64
	// Truncated is true when the result had more than MaxRows rows.
	Truncated bool
}

func New(connURL string) (*Client, error) {
	db, err := sql.Open("postgres", connURL)
	if err != nil {
		return nil, fmt.Errorf("open connection: %s", RedactURL(err.Error()))
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping: %s", RedactURL(err.Error()))
	}
	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(2)
	return &Client{db: db, connURL: connURL}, nil
}

var urlPassword = regexp.MustCompile(`(://[^:/@\s]*:)[^@\s]*@`)

// RedactURL masks the password of any connection URL inside s.
func RedactURL(s string) string {
	return urlPassword.ReplaceAllString(s, "${1}xxxxx@")
}

// conn borrows a connection from the pool, failing with ErrPoolExhausted
// instead of waiting indefinitely.
func (c *Client) conn(ctx context.Context) (*sql.Conn, error) {
	actx, cancel := context.WithTimeout(ctx, acquireTimeout)
	defer cancel()
	conn, err := c.db.Conn(actx)
	if err != nil && ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
		return nil, ErrPoolExhausted
	}
	return conn, err
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

func (c *Client) Info(ctx context.Context) (*ConnectionInfo, error) {
	info := &ConnectionInfo{}

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
	if err := c.db.QueryRowContext(ctx, "SHOW server_version").Scan(&version); err == nil {
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
	return New(u.String())
}

func (c *Client) Databases(ctx context.Context) ([]string, error) {
	rows, err := c.db.QueryContext(ctx, "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname")
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

// QueryWithContext runs user SQL on a pooled connection in autocommit mode.
// Session state the SQL changes (SET, SET ROLE, prepared statements) is
// discarded afterwards so it cannot leak into later queries from other tabs.
func (c *Client) QueryWithContext(ctx context.Context, query string) (*QueryResult, error) {
	conn, err := c.conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	result, err := runQuery(ctx, conn, query)
	if rerr := resetSession(conn); rerr != nil {
		if err == nil && isTxStateError(rerr) {
			err = ErrLeftTransactionOpen
		}
		if err != nil {
			return nil, err
		}
	}
	return result, err
}

// ErrLeftTransactionOpen is returned when autocommit SQL opens a transaction
// without closing it. The connection is dropped, which rolls it back.
var ErrLeftTransactionOpen = errors.New("the query opened a transaction without committing it, so it was rolled back; use the Begin button to keep a transaction open across runs")

// resetSession runs DISCARD ALL on conn. If that fails the connection is
// marked bad so the pool closes it rather than reusing it.
func resetSession(conn *sql.Conn) error {
	_, err := conn.ExecContext(context.Background(), "DISCARD ALL")
	if err != nil {
		conn.Raw(func(any) error { return driver.ErrBadConn })
	}
	return err
}

func isTxStateError(err error) bool {
	switch SQLState(err) {
	case "25001", "25P02":
		return true
	}
	return false
}

// queryContext runs internal, read-only SQL on a pooled connection.
func (c *Client) queryContext(ctx context.Context, query string) (*QueryResult, error) {
	conn, err := c.conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return runQuery(ctx, conn, query)
}

// runQuery executes a query and returns columns, types, rows and the command
// tag. It talks to the driver directly because database/sql hides the tag.
func runQuery(ctx context.Context, conn *sql.Conn, query string) (*QueryResult, error) {
	start := time.Now()
	var result *QueryResult
	err := conn.Raw(func(dc any) error {
		queryer, ok := dc.(driver.QueryerContext)
		if !ok {
			return errors.New("driver does not support QueryerContext")
		}
		rows, err := queryer.QueryContext(ctx, query, nil)
		if err != nil {
			return err
		}
		result, err = readRows(rows)
		if cerr := rows.Close(); err == nil {
			err = cerr
		}
		return err
	})
	if err != nil {
		return nil, withDetail(err)
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

// streamQuery runs a single-result query and hands each row to onRow
// without collecting them.
func streamQuery(ctx context.Context, conn *sql.Conn, query string, onHeader func(columns, types []string) error, onRow func([]*string) error) error {
	err := conn.Raw(func(dc any) error {
		queryer, ok := dc.(driver.QueryerContext)
		if !ok {
			return errors.New("driver does not support QueryerContext")
		}
		rows, err := queryer.QueryContext(ctx, query, nil)
		if err != nil {
			return err
		}
		defer rows.Close()
		columns := rows.Columns()
		typeNames := columnTypes(rows, len(columns))
		if err := onHeader(columns, typeNames); err != nil {
			return err
		}
		vals := make([]driver.Value, len(columns))
		for {
			if err := rows.Next(vals); err == io.EOF {
				return nil
			} else if err != nil {
				return err
			}
			if err := onRow(formatRow(vals, typeNames)); err != nil {
				return err
			}
		}
	})
	return withDetail(err)
}

func columnTypes(rows driver.Rows, n int) []string {
	typeNames := make([]string, n)
	if typed, ok := rows.(driver.RowsColumnTypeDatabaseTypeName); ok {
		for i := range typeNames {
			typeNames[i] = typed.ColumnTypeDatabaseTypeName(i)
		}
	}
	return typeNames
}

// detailedError carries PostgreSQL's DETAIL, HINT and error position, which
// usually say exactly what is wrong with a query.
type detailedError struct {
	msg string
	err error
}

func (e *detailedError) Error() string { return e.msg }
func (e *detailedError) Unwrap() error { return e.err }

func withDetail(err error) error {
	if err == nil {
		return nil
	}
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return err
	}
	return &detailedError{msg: pqErr.ErrorWithDetail(), err: err}
}

// SQLState returns the PostgreSQL error code of err, or "" if it has none.
func SQLState(err error) string {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code)
	}
	return ""
}

// readRows reads the first result set. The command tag is captured before the
// rows are closed so it describes that result set, not a later statement.
func readRows(rows driver.Rows) (*QueryResult, error) {
	// Non-nil so statements without a result set encode as [] rather than null.
	columns := append([]string{}, rows.Columns()...)
	typeNames := columnTypes(rows, len(columns))

	var data [][]*string
	truncated := false
	vals := make([]driver.Value, len(columns))
	for {
		if err := rows.Next(vals); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if len(data) == MaxRows {
			truncated = true
			continue
		}
		data = append(data, formatRow(vals, typeNames))
	}

	result := &QueryResult{
		Columns:     columns,
		ColumnTypes: typeNames,
		Rows:        data,
		RowCount:    len(data),
		Truncated:   truncated,
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
