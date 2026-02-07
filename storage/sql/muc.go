package sql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/meszmate/xmpp-go/storage"
)

type mucStore struct{ s *Store }

func (m *mucStore) CreateRoom(ctx context.Context, room *storage.MUCRoom) error {
	_, err := m.s.db.ExecContext(ctx,
		"INSERT INTO muc_rooms (room_jid, name, description, subject, password, is_public, is_persistent, max_users) VALUES ("+m.s.phs(1, 8)+")",
		room.RoomJID, room.Name, room.Description, room.Subject, room.Password, room.Public, room.Persistent, room.MaxUsers,
	)
	if err != nil && isUniqueViolation(err) {
		return storage.ErrUserExists
	}
	return err
}

func (m *mucStore) GetRoom(ctx context.Context, roomJID string) (*storage.MUCRoom, error) {
	var room storage.MUCRoom
	err := m.s.db.QueryRowContext(ctx,
		"SELECT room_jid, name, description, subject, password, is_public, is_persistent, max_users FROM muc_rooms WHERE room_jid = "+m.s.ph(1),
		roomJID,
	).Scan(&room.RoomJID, &room.Name, &room.Description, &room.Subject, &room.Password, &room.Public, &room.Persistent, &room.MaxUsers)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (m *mucStore) UpdateRoom(ctx context.Context, room *storage.MUCRoom) error {
	res, err := m.s.db.ExecContext(ctx,
		"UPDATE muc_rooms SET name = "+m.s.ph(1)+", description = "+m.s.ph(2)+", subject = "+m.s.ph(3)+", password = "+m.s.ph(4)+", is_public = "+m.s.ph(5)+", is_persistent = "+m.s.ph(6)+", max_users = "+m.s.ph(7)+" WHERE room_jid = "+m.s.ph(8),
		room.Name, room.Description, room.Subject, room.Password, room.Public, room.Persistent, room.MaxUsers, room.RoomJID,
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

func (m *mucStore) DeleteRoom(ctx context.Context, roomJID string) error {
	res, err := m.s.db.ExecContext(ctx, "DELETE FROM muc_rooms WHERE room_jid = "+m.s.ph(1), roomJID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return storage.ErrNotFound
	}
	// Also clean up affiliations.
	_, _ = m.s.db.ExecContext(ctx, "DELETE FROM muc_affiliations WHERE room_jid = "+m.s.ph(1), roomJID)
	return nil
}

func (m *mucStore) ListRooms(ctx context.Context) ([]*storage.MUCRoom, error) {
	rows, err := m.s.db.QueryContext(ctx, "SELECT room_jid, name, description, subject, password, is_public, is_persistent, max_users FROM muc_rooms")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*storage.MUCRoom
	for rows.Next() {
		var room storage.MUCRoom
		if err := rows.Scan(&room.RoomJID, &room.Name, &room.Description, &room.Subject, &room.Password, &room.Public, &room.Persistent, &room.MaxUsers); err != nil {
			return nil, err
		}
		rooms = append(rooms, &room)
	}
	return rooms, rows.Err()
}

func (m *mucStore) SetAffiliation(ctx context.Context, aff *storage.MUCAffiliation) error {
	q := "INSERT INTO muc_affiliations (room_jid, user_jid, affiliation, reason) VALUES (" + m.s.phs(1, 4) + ") " +
		m.s.dialect.UpsertSuffix([]string{"room_jid", "user_jid"}, []string{"affiliation", "reason"})
	_, err := m.s.db.ExecContext(ctx, q, aff.RoomJID, aff.UserJID, aff.Affiliation, aff.Reason)
	return err
}

func (m *mucStore) GetAffiliation(ctx context.Context, roomJID, userJID string) (*storage.MUCAffiliation, error) {
	var aff storage.MUCAffiliation
	err := m.s.db.QueryRowContext(ctx,
		"SELECT room_jid, user_jid, affiliation, reason FROM muc_affiliations WHERE room_jid = "+m.s.ph(1)+" AND user_jid = "+m.s.ph(2),
		roomJID, userJID,
	).Scan(&aff.RoomJID, &aff.UserJID, &aff.Affiliation, &aff.Reason)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &aff, nil
}

func (m *mucStore) GetAffiliations(ctx context.Context, roomJID string) ([]*storage.MUCAffiliation, error) {
	rows, err := m.s.db.QueryContext(ctx,
		"SELECT room_jid, user_jid, affiliation, reason FROM muc_affiliations WHERE room_jid = "+m.s.ph(1), roomJID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var affs []*storage.MUCAffiliation
	for rows.Next() {
		var aff storage.MUCAffiliation
		if err := rows.Scan(&aff.RoomJID, &aff.UserJID, &aff.Affiliation, &aff.Reason); err != nil {
			return nil, err
		}
		affs = append(affs, &aff)
	}
	return affs, rows.Err()
}

func (m *mucStore) RemoveAffiliation(ctx context.Context, roomJID, userJID string) error {
	_, err := m.s.db.ExecContext(ctx,
		"DELETE FROM muc_affiliations WHERE room_jid = "+m.s.ph(1)+" AND user_jid = "+m.s.ph(2),
		roomJID, userJID,
	)
	return err
}
