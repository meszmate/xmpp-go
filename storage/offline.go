package storage

import (
	"context"
	"time"
)

// OfflineMessage represents a stored offline message.
type OfflineMessage struct {
	ID        string
	UserJID   string
	FromJID   string
	Data      []byte // raw XML stanza
	CreatedAt time.Time
}

// OfflineStore manages offline messages.
type OfflineStore interface {
	// StoreOfflineMessage stores an offline message for a user.
	StoreOfflineMessage(ctx context.Context, msg *OfflineMessage) error

	// GetOfflineMessages retrieves all offline messages for a user.
	GetOfflineMessages(ctx context.Context, userJID string) ([]*OfflineMessage, error)

	// DeleteOfflineMessages removes all offline messages for a user.
	DeleteOfflineMessages(ctx context.Context, userJID string) error

	// CountOfflineMessages returns the number of offline messages for a user.
	CountOfflineMessages(ctx context.Context, userJID string) (int, error)
}
