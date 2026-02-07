// Package sqlite provides a SQLite storage backend for xmpp-go.
package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	xmppsql "github.com/meszmate/xmpp-go/storage/sql"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDialect implements the SQL dialect for SQLite.
type SQLiteDialect struct{}

func (d SQLiteDialect) Name() string          { return "sqlite" }
func (d SQLiteDialect) Placeholder(_ int) string { return "?" }
func (d SQLiteDialect) AutoIncrement() string { return "INTEGER PRIMARY KEY AUTOINCREMENT" }
func (d SQLiteDialect) BlobType() string      { return "BLOB" }
func (d SQLiteDialect) TimestampType() string { return "DATETIME" }
func (d SQLiteDialect) TextType() string      { return "TEXT" }
func (d SQLiteDialect) Now() string           { return "datetime('now')" }

func (d SQLiteDialect) UpsertSuffix(conflictColumns []string, updateColumns []string) string {
	if len(updateColumns) == 0 {
		return "ON CONFLICT (" + strings.Join(conflictColumns, ", ") + ") DO NOTHING"
	}
	sets := make([]string, len(updateColumns))
	for i, col := range updateColumns {
		sets[i] = col + " = excluded." + col
	}
	return "ON CONFLICT (" + strings.Join(conflictColumns, ", ") + ") DO UPDATE SET " + strings.Join(sets, ", ")
}

func (d SQLiteDialect) Migrations() []string {
	return sqliteMigrations
}

// New creates a new SQLite-backed storage.
func New(dsn string) (*xmppsql.Store, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	// Enable WAL mode and foreign keys.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: set WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: enable foreign keys: %w", err)
	}
	return xmppsql.New(db, SQLiteDialect{}), nil
}

var sqliteMigrations = []string{
	// Migration 1: users table
	`CREATE TABLE IF NOT EXISTS users (
		username TEXT PRIMARY KEY,
		password TEXT NOT NULL DEFAULT '',
		salt TEXT NOT NULL DEFAULT '',
		iterations INTEGER NOT NULL DEFAULT 0,
		server_key TEXT NOT NULL DEFAULT '',
		stored_key TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT (datetime('now')),
		updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`,

	// Migration 2: roster tables
	`CREATE TABLE IF NOT EXISTS roster_items (
		user_jid TEXT NOT NULL,
		contact_jid TEXT NOT NULL,
		name TEXT NOT NULL DEFAULT '',
		subscription TEXT NOT NULL DEFAULT 'none',
		ask TEXT NOT NULL DEFAULT '',
		groups_list TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (user_jid, contact_jid)
	);
	CREATE TABLE IF NOT EXISTS roster_versions (
		user_jid TEXT PRIMARY KEY,
		version TEXT NOT NULL DEFAULT ''
	)`,

	// Migration 3: blocked JIDs
	`CREATE TABLE IF NOT EXISTS blocked_jids (
		user_jid TEXT NOT NULL,
		blocked_jid TEXT NOT NULL,
		PRIMARY KEY (user_jid, blocked_jid)
	)`,

	// Migration 4: vcards
	`CREATE TABLE IF NOT EXISTS vcards (
		user_jid TEXT PRIMARY KEY,
		data BLOB NOT NULL
	)`,

	// Migration 5: offline messages
	`CREATE TABLE IF NOT EXISTS offline_messages (
		id TEXT NOT NULL,
		user_jid TEXT NOT NULL,
		from_jid TEXT NOT NULL DEFAULT '',
		data BLOB NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_offline_messages_user ON offline_messages(user_jid)`,

	// Migration 6: MAM messages
	`CREATE TABLE IF NOT EXISTS mam_messages (
		id TEXT NOT NULL,
		user_jid TEXT NOT NULL,
		with_jid TEXT NOT NULL DEFAULT '',
		from_jid TEXT NOT NULL DEFAULT '',
		data BLOB NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_mam_messages_user ON mam_messages(user_jid);
	CREATE INDEX IF NOT EXISTS idx_mam_messages_user_with ON mam_messages(user_jid, with_jid)`,

	// Migration 7: MUC rooms and affiliations
	`CREATE TABLE IF NOT EXISTS muc_rooms (
		room_jid TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		subject TEXT NOT NULL DEFAULT '',
		password TEXT NOT NULL DEFAULT '',
		is_public INTEGER NOT NULL DEFAULT 1,
		is_persistent INTEGER NOT NULL DEFAULT 0,
		max_users INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS muc_affiliations (
		room_jid TEXT NOT NULL,
		user_jid TEXT NOT NULL,
		affiliation TEXT NOT NULL DEFAULT 'none',
		reason TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (room_jid, user_jid)
	)`,

	// Migration 8: PubSub
	`CREATE TABLE IF NOT EXISTS pubsub_nodes (
		host TEXT NOT NULL,
		node_id TEXT NOT NULL,
		name TEXT NOT NULL DEFAULT '',
		type TEXT NOT NULL DEFAULT 'leaf',
		creator TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (host, node_id)
	);
	CREATE TABLE IF NOT EXISTS pubsub_items (
		host TEXT NOT NULL,
		node_id TEXT NOT NULL,
		item_id TEXT NOT NULL,
		publisher TEXT NOT NULL DEFAULT '',
		payload BLOB,
		created_at DATETIME NOT NULL DEFAULT (datetime('now')),
		PRIMARY KEY (host, node_id, item_id)
	);
	CREATE TABLE IF NOT EXISTS pubsub_subscriptions (
		host TEXT NOT NULL,
		node_id TEXT NOT NULL,
		jid TEXT NOT NULL,
		sub_id TEXT NOT NULL DEFAULT '',
		state TEXT NOT NULL DEFAULT 'subscribed',
		PRIMARY KEY (host, node_id, jid)
	)`,

	// Migration 9: Bookmarks
	`CREATE TABLE IF NOT EXISTS bookmarks (
		user_jid TEXT NOT NULL,
		room_jid TEXT NOT NULL,
		name TEXT NOT NULL DEFAULT '',
		nick TEXT NOT NULL DEFAULT '',
		password TEXT NOT NULL DEFAULT '',
		autojoin INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (user_jid, room_jid)
	)`,
}
