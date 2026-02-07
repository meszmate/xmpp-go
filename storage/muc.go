package storage

import "context"

// MUCRoom represents a multi-user chat room.
type MUCRoom struct {
	RoomJID     string
	Name        string
	Description string
	Subject     string
	Password    string
	Public      bool
	Persistent  bool
	MaxUsers    int
}

// MUCAffiliation represents a user's affiliation with a MUC room.
type MUCAffiliation struct {
	RoomJID     string
	UserJID     string
	Affiliation string // owner, admin, member, outcast, none
	Reason      string
}

// MUCRoomStore manages MUC room data.
type MUCRoomStore interface {
	// CreateRoom creates a new MUC room.
	CreateRoom(ctx context.Context, room *MUCRoom) error

	// GetRoom retrieves a MUC room by JID.
	GetRoom(ctx context.Context, roomJID string) (*MUCRoom, error)

	// UpdateRoom updates a MUC room.
	UpdateRoom(ctx context.Context, room *MUCRoom) error

	// DeleteRoom deletes a MUC room.
	DeleteRoom(ctx context.Context, roomJID string) error

	// ListRooms retrieves all rooms.
	ListRooms(ctx context.Context) ([]*MUCRoom, error)

	// SetAffiliation sets a user's affiliation in a room.
	SetAffiliation(ctx context.Context, aff *MUCAffiliation) error

	// GetAffiliation retrieves a user's affiliation in a room.
	GetAffiliation(ctx context.Context, roomJID, userJID string) (*MUCAffiliation, error)

	// GetAffiliations retrieves all affiliations for a room.
	GetAffiliations(ctx context.Context, roomJID string) ([]*MUCAffiliation, error)

	// RemoveAffiliation removes a user's affiliation from a room.
	RemoveAffiliation(ctx context.Context, roomJID, userJID string) error
}
