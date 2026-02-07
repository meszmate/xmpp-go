package storage

import "context"

// BlockingStore manages JID blocking lists.
type BlockingStore interface {
	// BlockJID adds a JID to the user's block list.
	BlockJID(ctx context.Context, userJID, blockedJID string) error

	// UnblockJID removes a JID from the user's block list.
	UnblockJID(ctx context.Context, userJID, blockedJID string) error

	// IsBlocked checks whether a JID is blocked by the user.
	IsBlocked(ctx context.Context, userJID, blockedJID string) (bool, error)

	// GetBlockedJIDs retrieves all blocked JIDs for a user.
	GetBlockedJIDs(ctx context.Context, userJID string) ([]string, error)
}
