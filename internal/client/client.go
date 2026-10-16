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

	"github.com/lib/pq"
)

type Client struct {
	db      *sql.DB
	connURL string
}

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
	Rows        [][]any
	RowCount    int
	DurationMs  int64
	// CommandTag is the verb of the final command, e.g. "UPDATE".
	CommandTag string
	// RowsAffected is nil for commands that do not report a count (DDL, BEGIN, ...).
	RowsAffected *int64
}

func New(connURL string) (*Client, error) {
	db, err := sql.Open("postgres", connURL)
	if err != nil {
		return nil, fmt.Errorf("open connection: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	return &Client{db: db, connURL: connURL}, nil
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
	return New(u.String())
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
	return c.queryContext(ctx, query)
}

// queryContext executes a query on a connection borrowed from the shared pool.
func (c *Client) queryContext(ctx context.Context, query string) (*QueryResult, error) {
	conn, err := c.db.Conn(ctx)
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

// detailedError carries PostgreSQL's DETAIL, HINT and error position, which
// usually say exactly what is wrong with a query.
type detailedError struct {
	msg string
	err error
}

func (e *detailedError) Error() string { return e.msg }
func (e *detailedError) Unwrap() error { return e.err }

func withDetail(err error) error {
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
	typeNames := make([]string, len(columns))
	if typed, ok := rows.(driver.RowsColumnTypeDatabaseTypeName); ok {
		for i := range columns {
			typeNames[i] = typed.ColumnTypeDatabaseTypeName(i)
		}
	}

	var data [][]any
	for {
		vals := make([]driver.Value, len(columns))
		if err := rows.Next(vals); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
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
