package client

import "context"

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
