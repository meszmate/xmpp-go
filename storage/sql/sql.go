// Package sql provides a shared SQL storage implementation for xmpp-go.
package sql

import (
	"context"
	"database/sql"

	"github.com/meszmate/xmpp-go/storage"
)

// Store implements storage.Storage using database/sql.
type Store struct {
	db      *sql.DB
	dialect Dialect
}

// New creates a new SQL-backed store.
func New(db *sql.DB, dialect Dialect) *Store {
	return &Store{db: db, dialect: dialect}
}

func (s *Store) Init(ctx context.Context) error {
	return Migrate(ctx, s.db, s.dialect)
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) UserStore() storage.UserStore         { return &userStore{s} }
func (s *Store) RosterStore() storage.RosterStore     { return &rosterStore{s} }
func (s *Store) BlockingStore() storage.BlockingStore { return &blockingStore{s} }
func (s *Store) VCardStore() storage.VCardStore       { return &vcardStore{s} }
func (s *Store) OfflineStore() storage.OfflineStore   { return &offlineStore{s} }
func (s *Store) MAMStore() storage.MAMStore           { return &mamStore{s} }
func (s *Store) MUCRoomStore() storage.MUCRoomStore   { return &mucStore{s} }
func (s *Store) PubSubStore() storage.PubSubStore     { return &pubsubStore{s} }
func (s *Store) BookmarkStore() storage.BookmarkStore { return &bookmarkStore{s} }

// ph is a helper that returns placeholders for the dialect.
func (s *Store) ph(n int) string {
	return s.dialect.Placeholder(n)
}

// phs returns a comma-separated list of placeholders.
func (s *Store) phs(start, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		if i > 0 {
			result += ", "
		}
		result += s.dialect.Placeholder(start + i)
	}
	return result
}
