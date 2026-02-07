package storage

import "context"

// RosterItem represents a roster entry.
type RosterItem struct {
	UserJID      string
	ContactJID   string
	Name         string
	Subscription string
	Ask          string
	Groups       []string
}

// RosterStore manages roster items.
type RosterStore interface {
	// UpsertRosterItem adds or updates a roster item.
	UpsertRosterItem(ctx context.Context, item *RosterItem) error

	// GetRosterItem retrieves a single roster item.
	GetRosterItem(ctx context.Context, userJID, contactJID string) (*RosterItem, error)

	// GetRosterItems retrieves all roster items for a user.
	GetRosterItems(ctx context.Context, userJID string) ([]*RosterItem, error)

	// DeleteRosterItem removes a roster item.
	DeleteRosterItem(ctx context.Context, userJID, contactJID string) error

	// GetRosterVersion retrieves the roster version for a user.
	GetRosterVersion(ctx context.Context, userJID string) (string, error)

	// SetRosterVersion sets the roster version for a user.
	SetRosterVersion(ctx context.Context, userJID, version string) error
}
