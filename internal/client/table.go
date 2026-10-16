package client

import (
	"context"
	"fmt"
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
	References   string // schema.table.column of the foreign key target
	EnumValues   []string
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
