package client

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
