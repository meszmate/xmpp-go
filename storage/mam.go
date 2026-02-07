package storage

import (
	"context"
	"time"
)

// ArchivedMessage represents a message stored in the archive.
type ArchivedMessage struct {
	ID        string
	UserJID   string
	WithJID   string
	FromJID   string
	Data      []byte // raw XML stanza
	CreatedAt time.Time
}

// MAMQuery represents query parameters for message archive retrieval.
type MAMQuery struct {
	UserJID string
	WithJID string    // filter by correspondent
	Start   time.Time // filter: after this time
	End     time.Time // filter: before this time
	AfterID string    // RSM: after this message ID
	BeforeID string   // RSM: before this message ID
	Max     int       // maximum results (0 = backend default)
}

// MAMResult represents the result of a MAM query.
type MAMResult struct {
	Messages []*ArchivedMessage
	Complete bool   // true if no more results
	First    string // RSM: first ID in result set
	Last     string // RSM: last ID in result set
	Count    int    // total count (if available)
}

// MAMStore manages the message archive.
type MAMStore interface {
	// ArchiveMessage stores a message in the archive.
	ArchiveMessage(ctx context.Context, msg *ArchivedMessage) error

	// QueryMessages retrieves messages matching the query.
	QueryMessages(ctx context.Context, query *MAMQuery) (*MAMResult, error)

	// DeleteMessageArchive removes all archived messages for a user.
	DeleteMessageArchive(ctx context.Context, userJID string) error
}
