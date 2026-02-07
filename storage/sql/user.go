package sql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/meszmate/xmpp-go/storage"
)

type userStore struct{ s *Store }

func (u *userStore) CreateUser(ctx context.Context, user *storage.User) error {
	now := time.Now()
	_, err := u.s.db.ExecContext(ctx,
		"INSERT INTO users (username, password, salt, iterations, server_key, stored_key, created_at, updated_at) VALUES ("+u.s.phs(1, 8)+")",
		user.Username, user.Password, user.Salt, user.Iterations, user.ServerKey, user.StoredKey, now, now,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return storage.ErrUserExists
		}
		return err
	}
	return nil
}

func (u *userStore) GetUser(ctx context.Context, username string) (*storage.User, error) {
	row := u.s.db.QueryRowContext(ctx,
		"SELECT username, password, salt, iterations, server_key, stored_key, created_at, updated_at FROM users WHERE username = "+u.s.ph(1),
		username,
	)
	var user storage.User
	err := row.Scan(&user.Username, &user.Password, &user.Salt, &user.Iterations, &user.ServerKey, &user.StoredKey, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *userStore) UpdateUser(ctx context.Context, user *storage.User) error {
	now := time.Now()
	res, err := u.s.db.ExecContext(ctx,
		"UPDATE users SET password = "+u.s.ph(1)+", salt = "+u.s.ph(2)+", iterations = "+u.s.ph(3)+", server_key = "+u.s.ph(4)+", stored_key = "+u.s.ph(5)+", updated_at = "+u.s.ph(6)+" WHERE username = "+u.s.ph(7),
		user.Password, user.Salt, user.Iterations, user.ServerKey, user.StoredKey, now, user.Username,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (u *userStore) DeleteUser(ctx context.Context, username string) error {
	res, err := u.s.db.ExecContext(ctx, "DELETE FROM users WHERE username = "+u.s.ph(1), username)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (u *userStore) UserExists(ctx context.Context, username string) (bool, error) {
	var count int
	err := u.s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE username = "+u.s.ph(1), username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (u *userStore) Authenticate(ctx context.Context, username, password string) (bool, error) {
	var storedPassword string
	err := u.s.db.QueryRowContext(ctx, "SELECT password FROM users WHERE username = "+u.s.ph(1), username).Scan(&storedPassword)
	if errors.Is(err, sql.ErrNoRows) {
		return false, storage.ErrAuthFailed
	}
	if err != nil {
		return false, err
	}
	if storedPassword != password {
		return false, storage.ErrAuthFailed
	}
	return true, nil
}

// isUniqueViolation checks for unique constraint violation errors across dialects.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// SQLite: "UNIQUE constraint failed"
	// PostgreSQL: "duplicate key value violates unique constraint"
	// MySQL: "Duplicate entry"
	return contains(msg, "UNIQUE constraint failed") ||
		contains(msg, "duplicate key") ||
		contains(msg, "Duplicate entry")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
