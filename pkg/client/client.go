package client

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
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

type SchemaObject struct {
	Name    string
	Schema  string
	Type    string
	Comment string
}

type SchemaGroup struct {
	Tables            []SchemaObject
	Views             []SchemaObject
	MaterializedViews []SchemaObject
	Functions         []SchemaObject
	Sequences         []SchemaObject
	Types             []SchemaObject
}

type Column struct {
	Name         string
	Type         string
	Nullable     bool
	DefaultValue *string
	Position     int
	IsPrimaryKey bool
	Comment      string
}

type QueryResult struct {
	Columns     []string
	ColumnTypes []string
	Rows        [][]any
	RowCount    int
	DurationMs  int64
}

type TableInfo struct {
	TotalSize   string
	TableSize   string
	IndexSize   string
	RowEstimate int64
}

type TableIndex struct {
	Name       string
	Definition string
	IsUnique   bool
	IsPrimary  bool
}

type TableConstraint struct {
	Name       string
	Type       string
	Definition string
}

type Activity struct {
	PID           int
	Database      string
	User          string
	Application   string
	ClientAddr    string
	State         string
	Query         string
	Duration      string
	WaitEvent     string
	WaitEventType string
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

func (c *Client) Schemas() ([]string, error) {
	rows, err := c.db.Query(`
		SELECT schema_name FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY schema_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		schemas = append(schemas, name)
	}
	return schemas, rows.Err()
}

func (c *Client) Objects() (map[string]*SchemaGroup, error) {
	result := make(map[string]*SchemaGroup)

	ensureGroup := func(schema string) *SchemaGroup {
		if g, ok := result[schema]; ok {
			return g
		}
		g := &SchemaGroup{}
		result[schema] = g
		return g
	}

	// Tables and views
	rows, err := c.db.Query(`
		SELECT table_schema, table_name, table_type,
			COALESCE(obj_description((table_schema || '.' || table_name)::regclass), '')
		FROM information_schema.tables
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY table_schema, table_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var schema, name, tableType, comment string
		if err := rows.Scan(&schema, &name, &tableType, &comment); err != nil {
			return nil, err
		}
		g := ensureGroup(schema)
		obj := SchemaObject{Name: name, Schema: schema, Comment: comment}
		if tableType == "BASE TABLE" {
			obj.Type = "table"
			g.Tables = append(g.Tables, obj)
		} else if tableType == "VIEW" {
			obj.Type = "view"
			g.Views = append(g.Views, obj)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Materialized views
	mvRows, err := c.db.Query(`
		SELECT schemaname, matviewname
		FROM pg_matviews
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY schemaname, matviewname`)
	if err == nil {
		defer mvRows.Close()
		for mvRows.Next() {
			var schema, name string
			if err := mvRows.Scan(&schema, &name); err != nil {
				continue
			}
			g := ensureGroup(schema)
			g.MaterializedViews = append(g.MaterializedViews, SchemaObject{
				Name: name, Schema: schema, Type: "materialized_view",
			})
		}
	}

	// Functions
	fnRows, err := c.db.Query(`
		SELECT n.nspname, p.proname,
			CASE p.prokind WHEN 'f' THEN 'function' WHEN 'p' THEN 'procedure' ELSE 'function' END
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY n.nspname, p.proname`)
	if err == nil {
		defer fnRows.Close()
		for fnRows.Next() {
			var schema, name, kind string
			if err := fnRows.Scan(&schema, &name, &kind); err != nil {
				continue
			}
			g := ensureGroup(schema)
			g.Functions = append(g.Functions, SchemaObject{
				Name: name, Schema: schema, Type: kind,
			})
		}
	}

	// Sequences
	seqRows, err := c.db.Query(`
		SELECT sequence_schema, sequence_name
		FROM information_schema.sequences
		WHERE sequence_schema NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY sequence_schema, sequence_name`)
	if err == nil {
		defer seqRows.Close()
		for seqRows.Next() {
			var schema, name string
			if err := seqRows.Scan(&schema, &name); err != nil {
				continue
			}
			g := ensureGroup(schema)
			g.Sequences = append(g.Sequences, SchemaObject{
				Name: name, Schema: schema, Type: "sequence",
			})
		}
	}

	// Types
	typeRows, err := c.db.Query(`
		SELECT n.nspname, t.typname
		FROM pg_type t
		JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE t.typtype = 'e'
		AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY n.nspname, t.typname`)
	if err == nil {
		defer typeRows.Close()
		for typeRows.Next() {
			var schema, name string
			if err := typeRows.Scan(&schema, &name); err != nil {
				continue
			}
			g := ensureGroup(schema)
			g.Types = append(g.Types, SchemaObject{
				Name: name, Schema: schema, Type: "enum",
			})
		}
	}

	return result, nil
}

// AllTableColumns fetches column names and types for all user tables in one query.
// Returns a map keyed by "schema.table" with slices of column name+type pairs.
func (c *Client) AllTableColumns() (map[string][]Column, error) {
	rows, err := c.db.Query(`
		SELECT c.table_schema, c.table_name, c.column_name, c.data_type
		FROM information_schema.columns c
		JOIN information_schema.tables t
		  ON t.table_schema = c.table_schema AND t.table_name = c.table_name
		WHERE t.table_type = 'BASE TABLE'
		  AND c.table_schema NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY c.table_schema, c.table_name, c.ordinal_position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]Column)
	for rows.Next() {
		var schema, table, colName, colType string
		if err := rows.Scan(&schema, &table, &colName, &colType); err != nil {
			return nil, err
		}
		key := schema + "." + table
		result[key] = append(result[key], Column{Name: colName, Type: colType})
	}
	return result, rows.Err()
}

func (c *Client) TableColumns(table string) ([]Column, error) {
	schema, name := splitTableName(table)

	rows, err := c.db.Query(`
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES',
			COALESCE(c.column_default, ''),
			c.ordinal_position,
			COALESCE(
				(SELECT true FROM information_schema.key_column_usage k
				 JOIN information_schema.table_constraints tc
				   ON tc.constraint_name = k.constraint_name
				   AND tc.table_schema = k.table_schema
				 WHERE k.table_schema = c.table_schema
				   AND k.table_name = c.table_name
				   AND k.column_name = c.column_name
				   AND tc.constraint_type = 'PRIMARY KEY'
				 LIMIT 1),
				false
			),
			COALESCE(
				col_description((c.table_schema || '.' || c.table_name)::regclass, c.ordinal_position),
				''
			)
		FROM information_schema.columns c
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position`, schema, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []Column
	for rows.Next() {
		var col Column
		var defVal string
		if err := rows.Scan(&col.Name, &col.Type, &col.Nullable, &defVal, &col.Position, &col.IsPrimaryKey, &col.Comment); err != nil {
			return nil, err
		}
		if defVal != "" {
			col.DefaultValue = &defVal
		}
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

func (c *Client) TableRows(ctx context.Context, table string, limit, offset int, sortCol, sortOrd string) (*QueryResult, int, error) {
	schema, name := splitTableName(table)
	fqn := fmt.Sprintf("%s.%s", quoteIdent(schema), quoteIdent(name))

	// Get total count
	var total int
	if err := c.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", fqn)).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build query
	q := fmt.Sprintf("SELECT * FROM %s", fqn)
	if sortCol != "" {
		ord := "ASC"
		if strings.EqualFold(sortOrd, "desc") {
			ord = "DESC"
		}
		q += fmt.Sprintf(" ORDER BY %s %s", quoteIdent(sortCol), ord)
	}
	q += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	result, err := c.queryContext(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

func (c *Client) TableInfo(table string) (*TableInfo, error) {
	schema, name := splitTableName(table)
	fqn := fmt.Sprintf("%s.%s", quoteIdent(schema), quoteIdent(name))

	info := &TableInfo{}
	err := c.db.QueryRow(`
		SELECT
			pg_size_pretty(pg_total_relation_size($1::regclass)),
			pg_size_pretty(pg_table_size($1::regclass)),
			pg_size_pretty(pg_indexes_size($1::regclass)),
			COALESCE((SELECT reltuples::bigint FROM pg_class WHERE oid = $1::regclass), 0)
	`, fqn).Scan(&info.TotalSize, &info.TableSize, &info.IndexSize, &info.RowEstimate)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (c *Client) TableIndexes(table string) ([]TableIndex, error) {
	schema, name := splitTableName(table)

	rows, err := c.db.Query(`
		SELECT
			i.relname,
			pg_get_indexdef(i.oid),
			ix.indisunique,
			ix.indisprimary
		FROM pg_index ix
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = $1 AND t.relname = $2
		ORDER BY i.relname`, schema, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []TableIndex
	for rows.Next() {
		var idx TableIndex
		if err := rows.Scan(&idx.Name, &idx.Definition, &idx.IsUnique, &idx.IsPrimary); err != nil {
			return nil, err
		}
		indexes = append(indexes, idx)
	}
	return indexes, rows.Err()
}

func (c *Client) TableConstraints(table string) ([]TableConstraint, error) {
	schema, name := splitTableName(table)

	rows, err := c.db.Query(`
		SELECT
			con.conname,
			con.contype::text,
			pg_get_constraintdef(con.oid)
		FROM pg_constraint con
		JOIN pg_class t ON t.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = $1 AND t.relname = $2
		ORDER BY con.conname`, schema, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []TableConstraint
	for rows.Next() {
		var con TableConstraint
		if err := rows.Scan(&con.Name, &con.Type, &con.Definition); err != nil {
			return nil, err
		}
		constraints = append(constraints, con)
	}
	return constraints, rows.Err()
}

type FunctionDefinition struct {
	Name       string
	Schema     string
	Definition string
	Language   string
	Arguments  string
	ReturnType string
	Volatility string
	Kind       string
}

func (c *Client) FunctionDefinition(schema, name string) (*FunctionDefinition, error) {
	fd := &FunctionDefinition{}
	err := c.db.QueryRow(`
		SELECT p.proname, n.nspname, pg_get_functiondef(p.oid),
			l.lanname, pg_get_function_arguments(p.oid),
			COALESCE(pg_get_function_result(p.oid), ''),
			CASE p.provolatile WHEN 'i' THEN 'IMMUTABLE' WHEN 's' THEN 'STABLE' ELSE 'VOLATILE' END,
			CASE p.prokind WHEN 'p' THEN 'procedure' ELSE 'function' END
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		LEFT JOIN pg_language l ON l.oid = p.prolang
		WHERE n.nspname = $1 AND p.proname = $2
		LIMIT 1`, schema, name).Scan(
		&fd.Name, &fd.Schema, &fd.Definition, &fd.Language,
		&fd.Arguments, &fd.ReturnType, &fd.Volatility, &fd.Kind,
	)
	if err != nil {
		return nil, err
	}
	return fd, nil
}

func (c *Client) QueryWithContext(ctx context.Context, query string) (*QueryResult, error) {
	return c.queryContext(ctx, query)
}

func (c *Client) TablesStats() (*QueryResult, error) {
	return c.queryContext(context.Background(), `
		SELECT
			schemaname || '.' || relname AS table,
			n_live_tup AS rows,
			pg_size_pretty(pg_total_relation_size(schemaname || '.' || relname)) AS total_size,
			n_tup_ins AS inserts,
			n_tup_upd AS updates,
			n_tup_del AS deletes,
			last_vacuum,
			last_autovacuum,
			last_analyze,
			last_autoanalyze
		FROM pg_stat_user_tables
		ORDER BY n_live_tup DESC`)
}

func (c *Client) Activity() ([]Activity, error) {
	rows, err := c.db.Query(`
		SELECT
			pid,
			COALESCE(datname, ''),
			COALESCE(usename, ''),
			COALESCE(application_name, ''),
			COALESCE(client_addr::text, ''),
			COALESCE(state, ''),
			COALESCE(query, ''),
			COALESCE(now() - query_start, interval '0')::text,
			COALESCE(wait_event, ''),
			COALESCE(wait_event_type, '')
		FROM pg_stat_activity
		WHERE pid != pg_backend_pid()
		ORDER BY query_start DESC NULLS LAST`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []Activity
	for rows.Next() {
		var a Activity
		if err := rows.Scan(&a.PID, &a.Database, &a.User, &a.Application, &a.ClientAddr,
			&a.State, &a.Query, &a.Duration, &a.WaitEvent, &a.WaitEventType); err != nil {
			return nil, err
		}
		activities = append(activities, a)
	}
	return activities, rows.Err()
}

func (c *Client) ServerSettings() (*QueryResult, error) {
	return c.queryContext(context.Background(), `
		SELECT name, setting, unit, short_desc
		FROM pg_settings
		ORDER BY name`)
}

// queryContext executes a query and returns columns, types, and rows.
func (c *Client) queryContext(ctx context.Context, query string) (*QueryResult, error) {
	start := time.Now()
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	columns := make([]string, len(colTypes))
	typeNames := make([]string, len(colTypes))
	for i, ct := range colTypes {
		columns[i] = ct.Name()
		typeNames[i] = ct.DatabaseTypeName()
	}

	var data [][]any
	for rows.Next() {
		vals := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		// Convert []byte to string for readability
		for i, v := range vals {
			if b, ok := v.([]byte); ok {
				vals[i] = string(b)
			}
		}
		data = append(data, vals)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &QueryResult{
		Columns:     columns,
		ColumnTypes: typeNames,
		Rows:        data,
		RowCount:    len(data),
		DurationMs:  time.Since(start).Milliseconds(),
	}, nil
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
