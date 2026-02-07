package sql

import (
	"context"
	"database/sql"
	"fmt"
)

// Migrate runs all pending migrations.
func Migrate(ctx context.Context, db *sql.DB, dialect Dialect) error {
	// Create migrations tracking table.
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS xmpp_migrations (
		version INTEGER PRIMARY KEY,
		applied_at `+dialect.TimestampType()+` DEFAULT (`+dialect.Now()+`)
	)`)
	if err != nil {
		return fmt.Errorf("sql: create migrations table: %w", err)
	}

	migrations := dialect.Migrations()
	for i, m := range migrations {
		version := i + 1

		var count int
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM xmpp_migrations WHERE version = "+dialect.Placeholder(1), version).Scan(&count)
		if err != nil {
			return fmt.Errorf("sql: check migration %d: %w", version, err)
		}
		if count > 0 {
			continue
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("sql: begin migration %d: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, m); err != nil {
			tx.Rollback()
			return fmt.Errorf("sql: run migration %d: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT INTO xmpp_migrations (version) VALUES ("+dialect.Placeholder(1)+")", version); err != nil {
			tx.Rollback()
			return fmt.Errorf("sql: record migration %d: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("sql: commit migration %d: %w", version, err)
		}
	}

	return nil
}
