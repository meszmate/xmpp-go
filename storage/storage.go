// Package storage defines the pluggable storage interfaces for xmpp-go.
package storage

import (
	"context"
	"errors"
	"io"
)

// Sentinel errors for storage operations.
var (
	ErrNotFound   = errors.New("storage: not found")
	ErrUserExists = errors.New("storage: user already exists")
	ErrAuthFailed = errors.New("storage: authentication failed")
)

// Storage is the composite storage interface that provides access to all sub-stores.
type Storage interface {
	io.Closer

	// Init initializes the storage backend (e.g. create tables, open connections).
	Init(ctx context.Context) error

	// UserStore returns the user store, or nil if unsupported.
	UserStore() UserStore

	// RosterStore returns the roster store, or nil if unsupported.
	RosterStore() RosterStore

	// BlockingStore returns the blocking store, or nil if unsupported.
	BlockingStore() BlockingStore

	// VCardStore returns the vcard store, or nil if unsupported.
	VCardStore() VCardStore

	// OfflineStore returns the offline message store, or nil if unsupported.
	OfflineStore() OfflineStore

	// MAMStore returns the message archive store, or nil if unsupported.
	MAMStore() MAMStore

	// MUCRoomStore returns the MUC room store, or nil if unsupported.
	MUCRoomStore() MUCRoomStore

	// PubSubStore returns the pubsub store, or nil if unsupported.
	PubSubStore() PubSubStore

	// BookmarkStore returns the bookmark store, or nil if unsupported.
	BookmarkStore() BookmarkStore
}
