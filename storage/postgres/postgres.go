// Package postgres provides a PostgreSQL storage backend for xmpp-go.
package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	xmppsql "github.com/meszmate/xmpp-go/storage/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresDialect implements the SQL dialect for PostgreSQL.
type PostgresDialect struct{}

func (d PostgresDialect) Name() string { return "postgres" }

func (d PostgresDialect) Placeholder(n int) string {
	return fmt.Sprintf("$%d", n)
}

func (d PostgresDialect) AutoIncrement() string { return "BIGSERIAL PRIMARY KEY" }
func (d PostgresDialect) BlobType() string       { return "BYTEA" }
func (d PostgresDialect) TimestampType() string  { return "TIMESTAMPTZ" }
func (d PostgresDialect) TextType() string       { return "TEXT" }
func (d PostgresDialect) Now() string            { return "NOW()" }

func (d PostgresDialect) UpsertSuffix(conflictColumns []string, updateColumns []string) string {
	if len(updateColumns) == 0 {
		return "ON CONFLICT (" + strings.Join(conflictColumns, ", ") + ") DO NOTHING"
	}
	sets := make([]string, len(updateColumns))
	for i, col := range updateColumns {
		sets[i] = col + " = EXCLUDED." + col
	}
	return "ON CONFLICT (" + strings.Join(conflictColumns, ", ") + ") DO UPDATE SET " + strings.Join(sets, ", ")
}

func (d PostgresDialect) Migrations() []string {
	return postgresMigrations
}

// New creates a new PostgreSQL-backed storage.
func New(dsn string) (*xmppsql.Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: open: %w", err)
	}
	return xmppsql.New(db, PostgresDialect{}), nil
}

var postgresMigrations = []string{
	// Migration 1: users table
	`CREATE TABLE IF NOT EXISTS users (
		username TEXT PRIMARY KEY,
		password TEXT NOT NULL DEFAULT '',
		salt TEXT NOT NULL DEFAULT '',
		iterations INTEGER NOT NULL DEFAULT 0,
		server_key TEXT NOT NULL DEFAULT '',
		stored_key TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
		data BYTEA NOT NULL
	)`,

	// Migration 5: offline messages
	`CREATE TABLE IF NOT EXISTS offline_messages (
		id TEXT NOT NULL,
		user_jid TEXT NOT NULL,
		from_jid TEXT NOT NULL DEFAULT '',
		data BYTEA NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_offline_messages_user ON offline_messages(user_jid)`,

	// Migration 6: MAM messages
	`CREATE TABLE IF NOT EXISTS mam_messages (
		id TEXT NOT NULL,
		user_jid TEXT NOT NULL,
		with_jid TEXT NOT NULL DEFAULT '',
		from_jid TEXT NOT NULL DEFAULT '',
		data BYTEA NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
		is_public BOOLEAN NOT NULL DEFAULT TRUE,
		is_persistent BOOLEAN NOT NULL DEFAULT FALSE,
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
		payload BYTEA,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
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
		autojoin BOOLEAN NOT NULL DEFAULT FALSE,
		PRIMARY KEY (user_jid, room_jid)
	)`,
}
