package storage

import "context"

// VCardStore manages vCard data as raw XML blobs.
type VCardStore interface {
	// SetVCard stores a vCard XML blob for a user.
	SetVCard(ctx context.Context, userJID string, data []byte) error

	// GetVCard retrieves a vCard XML blob for a user.
	GetVCard(ctx context.Context, userJID string) ([]byte, error)

	// DeleteVCard removes a vCard for a user.
	DeleteVCard(ctx context.Context, userJID string) error
}
