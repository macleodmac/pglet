package client

import (
	"context"

	"github.com/lib/pq"
)

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

// Relation is a table, view, materialized view or foreign table.
type Relation struct {
	Schema  string
	Name    string
	Kind    string
	Comment string
	Columns []Column
}

// Relations describes every user table, view, materialized view and foreign
// table, with primary keys, foreign key targets and enum values per column.
// Partitions are omitted in favour of their parent.
func (c *Client) Relations(ctx context.Context) ([]Relation, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT
			n.nspname,
			c.relname,
			CASE c.relkind
				WHEN 'v' THEN 'view'
				WHEN 'm' THEN 'materialized view'
				WHEN 'f' THEN 'foreign table'
				ELSE 'table'
			END,
			COALESCE(obj_description(c.oid, 'pg_class'), ''),
			a.attname,
			format_type(a.atttypid, a.atttypmod),
			COALESCE(col_description(c.oid, a.attnum), ''),
			EXISTS (
				SELECT 1 FROM pg_constraint pk
				WHERE pk.conrelid = c.oid AND pk.contype = 'p' AND a.attnum = ANY (pk.conkey)
			),
			COALESCE(fk.ref, ''),
			en.vals
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
		LEFT JOIN LATERAL (
			SELECT rn.nspname || '.' || rc.relname || '.' || ra.attname AS ref
			FROM pg_constraint con
			CROSS JOIN LATERAL unnest(con.conkey, con.confkey) AS k(col, refcol)
			JOIN pg_class rc ON rc.oid = con.confrelid
			JOIN pg_namespace rn ON rn.oid = rc.relnamespace
			JOIN pg_attribute ra ON ra.attrelid = con.confrelid AND ra.attnum = k.refcol
			WHERE con.conrelid = c.oid AND con.contype = 'f' AND k.col = a.attnum
			LIMIT 1
		) fk ON true
		LEFT JOIN LATERAL (
			SELECT array_agg(e.enumlabel ORDER BY e.enumsortorder) AS vals
			FROM pg_enum e
			WHERE e.enumtypid = a.atttypid
		) en ON true
		WHERE c.relkind IN ('r', 'p', 'v', 'm', 'f')
		  AND NOT c.relispartition
		  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
		  AND n.nspname NOT LIKE 'pg\_toast%'
		  AND n.nspname NOT LIKE 'pg\_temp%'
		ORDER BY n.nspname, c.relname, a.attnum`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []Relation
	for rows.Next() {
		var rel Relation
		var col Column
		if err := rows.Scan(&rel.Schema, &rel.Name, &rel.Kind, &rel.Comment,
			&col.Name, &col.Type, &col.Comment, &col.IsPrimaryKey, &col.References, pq.Array(&col.EnumValues)); err != nil {
			return nil, err
		}
		if n := len(rels); n == 0 || rels[n-1].Schema != rel.Schema || rels[n-1].Name != rel.Name {
			rels = append(rels, rel)
		}
		last := &rels[len(rels)-1]
		last.Columns = append(last.Columns, col)
	}
	return rels, rows.Err()
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
