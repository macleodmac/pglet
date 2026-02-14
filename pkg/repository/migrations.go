package repository

var migrations = []string{
	// 001: saved queries
	`CREATE TABLE IF NOT EXISTS saved_queries (
		id          TEXT PRIMARY KEY DEFAULT (hex(randomblob(8))),
		title       TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		sql         TEXT NOT NULL,
		database    TEXT NOT NULL DEFAULT '',
		tags        TEXT NOT NULL DEFAULT '',
		shared      INTEGER NOT NULL DEFAULT 0,
		created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);`,

	// 002: query history
	`CREATE TABLE IF NOT EXISTS query_history (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		sql         TEXT NOT NULL,
		database    TEXT NOT NULL DEFAULT '',
		duration_ms INTEGER NOT NULL DEFAULT 0,
		row_count   INTEGER NOT NULL DEFAULT 0,
		error       TEXT NOT NULL DEFAULT '',
		executed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);
	CREATE INDEX IF NOT EXISTS idx_history_executed_at ON query_history(executed_at DESC);`,

	// 003: settings
	`CREATE TABLE IF NOT EXISTS settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL DEFAULT ''
	);
	INSERT OR IGNORE INTO settings (key, value) VALUES ('ai_api_key', '');
	INSERT OR IGNORE INTO settings (key, value) VALUES ('ai_model', '');`,
}
