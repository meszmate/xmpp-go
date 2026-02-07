package storage

import "context"

// Bookmark represents a bookmarked conference room.
type Bookmark struct {
	UserJID  string
	RoomJID  string
	Name     string
	Nick     string
	Password string
	Autojoin bool
}

// BookmarkStore manages user bookmarks.
type BookmarkStore interface {
	// SetBookmark adds or updates a bookmark.
	SetBookmark(ctx context.Context, bm *Bookmark) error

	// GetBookmark retrieves a bookmark.
	GetBookmark(ctx context.Context, userJID, roomJID string) (*Bookmark, error)

	// GetBookmarks retrieves all bookmarks for a user.
	GetBookmarks(ctx context.Context, userJID string) ([]*Bookmark, error)

	// DeleteBookmark removes a bookmark.
	DeleteBookmark(ctx context.Context, userJID, roomJID string) error
}
