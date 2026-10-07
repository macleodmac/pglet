package client

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type Column struct {
	Name         string
	Type         string
	Nullable     bool
	DefaultValue *string
	Position     int
	IsPrimaryKey bool
	Comment      string
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

// FilterOp is a comparison used by a table browser filter.
type FilterOp string

const (
	OpEq      FilterOp = "eq"
	OpNeq     FilterOp = "neq"
	OpLt      FilterOp = "lt"
	OpLte     FilterOp = "lte"
	OpGt      FilterOp = "gt"
	OpGte     FilterOp = "gte"
	OpLike    FilterOp = "like"
	OpILike   FilterOp = "ilike"
	OpIsNull  FilterOp = "is_null"
	OpNotNull FilterOp = "not_null"
)

var comparisonOps = map[FilterOp]string{
	OpEq: "=", OpNeq: "<>", OpLt: "<", OpLte: "<=", OpGt: ">", OpGte: ">=",
}

// Filter restricts table rows on one column. Value is bound as a parameter,
// so Postgres parses it as the column's type.
type Filter struct {
	Column string
	Op     FilterOp
	Value  string
}

// whereClause builds "WHERE ..." from filters, numbering parameters from $1.
func whereClause(filters []Filter) (string, []any, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}
	conds := make([]string, 0, len(filters))
	var args []any
	for _, f := range filters {
		col := quoteIdent(f.Column)
		switch f.Op {
		case OpIsNull:
			conds = append(conds, col+" IS NULL")
		case OpNotNull:
			conds = append(conds, col+" IS NOT NULL")
		case OpLike, OpILike:
			args = append(args, f.Value)
			// Cast so pattern matching works on any column type.
			conds = append(conds, fmt.Sprintf("%s::text %s $%d", col, strings.ToUpper(string(f.Op)), len(args)))
		default:
			op, ok := comparisonOps[f.Op]
			if !ok {
				return "", nil, fmt.Errorf("unknown filter operator %q", f.Op)
			}
			args = append(args, f.Value)
			conds = append(conds, fmt.Sprintf("%s %s $%d", col, op, len(args)))
		}
	}
	return " WHERE " + strings.Join(conds, " AND "), args, nil
}

func (c *Client) TableRows(ctx context.Context, table string, limit, offset int, sortCol, sortOrd string, filters []Filter) (*QueryResult, int, error) {
	schema, name := splitTableName(table)
	fqn := fmt.Sprintf("%s.%s", quoteIdent(schema), quoteIdent(name))

	where, args, err := whereClause(filters)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	var total int
	if err := c.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s%s", fqn, where), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build query
	q := fmt.Sprintf("SELECT * FROM %s%s", fqn, where)
	if sortCol != "" {
		ord := "ASC"
		if strings.EqualFold(sortOrd, "desc") {
			ord = "DESC"
		}
		q += fmt.Sprintf(" ORDER BY %s %s", quoteIdent(sortCol), ord)
	}
	q += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	result, err := c.queryContext(ctx, q, queryOpts{args: args})
	if err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

// ErrRowNotUnique is returned by UpdateRow when the key would not match
// exactly one row, so nothing was changed.
var ErrRowNotUnique = errors.New("row update must match exactly one row")

// UpdateRow sets columns on the single row identified by its primary key.
// key must name every primary key column. It runs in its own transaction and
// rolls back unless exactly one row changed. A nil value means NULL.
func (c *Client) UpdateRow(ctx context.Context, table string, key, changes map[string]*string) error {
	if len(changes) == 0 {
		return errors.New("no changes to apply")
	}
	cols, err := c.TableColumns(table)
	if err != nil {
		return err
	}
	var pk []string
	for _, col := range cols {
		if col.IsPrimaryKey {
			pk = append(pk, col.Name)
		}
	}
	if len(pk) == 0 {
		return errors.New("table has no primary key, so rows cannot be edited")
	}
	if len(key) != len(pk) {
		return fmt.Errorf("key must name exactly the primary key columns: %s", strings.Join(pk, ", "))
	}
	for _, name := range pk {
		if v, ok := key[name]; !ok || v == nil {
			return fmt.Errorf("key is missing primary key column %q", name)
		}
	}

	schema, name := splitTableName(table)
	fqn := fmt.Sprintf("%s.%s", quoteIdent(schema), quoteIdent(name))

	var args []any
	param := func(v *string) string {
		if v == nil {
			args = append(args, nil)
		} else {
			args = append(args, *v)
		}
		return fmt.Sprintf("$%d", len(args))
	}

	changed := sortedKeys(changes)
	sets := make([]string, len(changed))
	for i, col := range changed {
		sets[i] = fmt.Sprintf("%s = %s", quoteIdent(col), param(changes[col]))
	}
	conds := make([]string, len(pk))
	for i, col := range pk {
		conds[i] = fmt.Sprintf("%s = %s", quoteIdent(col), param(key[col]))
	}
	q := fmt.Sprintf("UPDATE %s SET %s WHERE %s", fqn, strings.Join(sets, ", "), strings.Join(conds, " AND "))

	// database/sql's default BeginTx makes lib/pq send BEGIN READ WRITE, which
	// would override the read-only option.
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: c.opts.ReadOnly})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w (matched %d)", ErrRowNotUnique, n)
	}
	return tx.Commit()
}

func sortedKeys(m map[string]*string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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
