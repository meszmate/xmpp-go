package sql

// Dialect abstracts database-specific SQL differences.
type Dialect interface {
	// Name returns the dialect name (e.g. "sqlite", "postgres", "mysql").
	Name() string

	// Placeholder returns the parameter placeholder for the nth parameter (1-indexed).
	// e.g. SQLite/MySQL return "?", PostgreSQL returns "$1", "$2", etc.
	Placeholder(n int) string

	// AutoIncrement returns the column type for an auto-incrementing primary key.
	AutoIncrement() string

	// Migrations returns the SQL migration statements for this dialect.
	Migrations() []string

	// UpsertSuffix returns the dialect-specific upsert clause.
	// For SQLite: "ON CONFLICT(...) DO UPDATE SET ..."
	// For PostgreSQL: "ON CONFLICT(...) DO UPDATE SET ..."
	// For MySQL: "ON DUPLICATE KEY UPDATE ..."
	UpsertSuffix(conflictColumns []string, updateColumns []string) string

	// BlobType returns the column type for binary data.
	BlobType() string

	// TimestampType returns the column type for timestamps.
	TimestampType() string

	// TextType returns the column type for text data.
	TextType() string

	// Now returns the SQL expression for the current timestamp.
	Now() string
}
