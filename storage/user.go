package storage

import (
	"context"
	"time"
)

// User represents a stored user account.
type User struct {
	Username  string
	Password  string // plaintext or hashed, depending on backend
	Salt      string // SCRAM salt (base64-encoded)
	Iterations int   // SCRAM iteration count
	ServerKey string // SCRAM server key (base64-encoded)
	StoredKey string // SCRAM stored key (base64-encoded)
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserStore manages user accounts.
type UserStore interface {
	// CreateUser creates a new user account.
	CreateUser(ctx context.Context, user *User) error

	// GetUser retrieves a user by username.
	GetUser(ctx context.Context, username string) (*User, error)

	// UpdateUser updates an existing user account.
	UpdateUser(ctx context.Context, user *User) error

	// DeleteUser deletes a user account.
	DeleteUser(ctx context.Context, username string) error

	// UserExists checks whether a user exists.
	UserExists(ctx context.Context, username string) (bool, error)

	// Authenticate validates username and password. Returns ErrAuthFailed on mismatch.
	Authenticate(ctx context.Context, username, password string) (bool, error)
}
