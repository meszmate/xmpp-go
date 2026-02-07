// Package mysql provides a MySQL storage backend for xmpp-go.
package mysql

import (
	"database/sql"
	"fmt"
	"strings"

	xmppsql "github.com/meszmate/xmpp-go/storage/sql"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLDialect implements the SQL dialect for MySQL.
type MySQLDialect struct{}

func (d MySQLDialect) Name() string            { return "mysql" }
func (d MySQLDialect) Placeholder(_ int) string { return "?" }
func (d MySQLDialect) AutoIncrement() string   { return "BIGINT PRIMARY KEY AUTO_INCREMENT" }
func (d MySQLDialect) BlobType() string        { return "LONGBLOB" }
func (d MySQLDialect) TimestampType() string   { return "DATETIME(6)" }
func (d MySQLDialect) TextType() string        { return "VARCHAR(512)" }
func (d MySQLDialect) Now() string             { return "NOW(6)" }

func (d MySQLDialect) UpsertSuffix(conflictColumns []string, updateColumns []string) string {
	_ = conflictColumns // MySQL uses ON DUPLICATE KEY, no conflict columns needed
	if len(updateColumns) == 0 {
		// MySQL has no DO NOTHING; use a no-op update on first conflict column.
		return "ON DUPLICATE KEY UPDATE " + conflictColumns[0] + " = " + conflictColumns[0]
	}
	sets := make([]string, len(updateColumns))
	for i, col := range updateColumns {
		sets[i] = col + " = VALUES(" + col + ")"
	}
	return "ON DUPLICATE KEY UPDATE " + strings.Join(sets, ", ")
}

func (d MySQLDialect) Migrations() []string {
	return mysqlMigrations
}

// New creates a new MySQL-backed storage.
func New(dsn string) (*xmppsql.Store, error) {
	db, err := sql.Open("mysql", dsn+"?parseTime=true")
	if err != nil {
		return nil, fmt.Errorf("mysql: open: %w", err)
	}
	return xmppsql.New(db, MySQLDialect{}), nil
}

var mysqlMigrations = []string{
	// Migration 1: users table
	`CREATE TABLE IF NOT EXISTS users (
		username VARCHAR(512) PRIMARY KEY,
		password TEXT NOT NULL,
		salt TEXT NOT NULL,
		iterations INT NOT NULL DEFAULT 0,
		server_key TEXT NOT NULL,
		stored_key TEXT NOT NULL,
		created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
		updated_at DATETIME(6) NOT NULL DEFAULT NOW(6)
	)`,

	// Migration 2: roster tables
	`CREATE TABLE IF NOT EXISTS roster_items (
		user_jid VARCHAR(512) NOT NULL,
		contact_jid VARCHAR(512) NOT NULL,
		name TEXT NOT NULL,
		subscription VARCHAR(32) NOT NULL DEFAULT 'none',
		ask VARCHAR(32) NOT NULL DEFAULT '',
		groups_list TEXT NOT NULL,
		PRIMARY KEY (user_jid, contact_jid)
	)`,

	// Migration 2b: roster versions
	`CREATE TABLE IF NOT EXISTS roster_versions (
		user_jid VARCHAR(512) PRIMARY KEY,
		version TEXT NOT NULL
	)`,

	// Migration 3: blocked JIDs
	`CREATE TABLE IF NOT EXISTS blocked_jids (
		user_jid VARCHAR(512) NOT NULL,
		blocked_jid VARCHAR(512) NOT NULL,
		PRIMARY KEY (user_jid, blocked_jid)
	)`,

	// Migration 4: vcards
	`CREATE TABLE IF NOT EXISTS vcards (
		user_jid VARCHAR(512) PRIMARY KEY,
		data LONGBLOB NOT NULL
	)`,

	// Migration 5: offline messages
	`CREATE TABLE IF NOT EXISTS offline_messages (
		id VARCHAR(512) NOT NULL,
		user_jid VARCHAR(512) NOT NULL,
		from_jid VARCHAR(512) NOT NULL DEFAULT '',
		data LONGBLOB NOT NULL,
		created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
		INDEX idx_offline_messages_user (user_jid)
	)`,

	// Migration 6: MAM messages
	`CREATE TABLE IF NOT EXISTS mam_messages (
		id VARCHAR(512) NOT NULL,
		user_jid VARCHAR(512) NOT NULL,
		with_jid VARCHAR(512) NOT NULL DEFAULT '',
		from_jid VARCHAR(512) NOT NULL DEFAULT '',
		data LONGBLOB NOT NULL,
		created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
		INDEX idx_mam_messages_user (user_jid),
		INDEX idx_mam_messages_user_with (user_jid, with_jid)
	)`,

	// Migration 7: MUC rooms
	`CREATE TABLE IF NOT EXISTS muc_rooms (
		room_jid VARCHAR(512) PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		subject TEXT NOT NULL,
		password TEXT NOT NULL,
		is_public BOOLEAN NOT NULL DEFAULT TRUE,
		is_persistent BOOLEAN NOT NULL DEFAULT FALSE,
		max_users INT NOT NULL DEFAULT 0
	)`,

	// Migration 7b: MUC affiliations
	`CREATE TABLE IF NOT EXISTS muc_affiliations (
		room_jid VARCHAR(512) NOT NULL,
		user_jid VARCHAR(512) NOT NULL,
		affiliation VARCHAR(32) NOT NULL DEFAULT 'none',
		reason TEXT NOT NULL,
		PRIMARY KEY (room_jid, user_jid)
	)`,

	// Migration 8: PubSub nodes
	`CREATE TABLE IF NOT EXISTS pubsub_nodes (
		host VARCHAR(512) NOT NULL,
		node_id VARCHAR(512) NOT NULL,
		name TEXT NOT NULL,
		type VARCHAR(32) NOT NULL DEFAULT 'leaf',
		creator VARCHAR(512) NOT NULL DEFAULT '',
		PRIMARY KEY (host, node_id)
	)`,

	// Migration 8b: PubSub items
	`CREATE TABLE IF NOT EXISTS pubsub_items (
		host VARCHAR(512) NOT NULL,
		node_id VARCHAR(512) NOT NULL,
		item_id VARCHAR(512) NOT NULL,
		publisher VARCHAR(512) NOT NULL DEFAULT '',
		payload LONGBLOB,
		created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
		PRIMARY KEY (host, node_id, item_id)
	)`,

	// Migration 8c: PubSub subscriptions
	`CREATE TABLE IF NOT EXISTS pubsub_subscriptions (
		host VARCHAR(512) NOT NULL,
		node_id VARCHAR(512) NOT NULL,
		jid VARCHAR(512) NOT NULL,
		sub_id VARCHAR(512) NOT NULL DEFAULT '',
		state VARCHAR(32) NOT NULL DEFAULT 'subscribed',
		PRIMARY KEY (host, node_id, jid)
	)`,

	// Migration 9: Bookmarks
	`CREATE TABLE IF NOT EXISTS bookmarks (
		user_jid VARCHAR(512) NOT NULL,
		room_jid VARCHAR(512) NOT NULL,
		name TEXT NOT NULL,
		nick TEXT NOT NULL,
		password TEXT NOT NULL,
		autojoin BOOLEAN NOT NULL DEFAULT FALSE,
		PRIMARY KEY (user_jid, room_jid)
	)`,
}
