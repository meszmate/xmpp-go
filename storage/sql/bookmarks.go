package sql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/meszmate/xmpp-go/storage"
)

type bookmarkStore struct{ s *Store }

func (b *bookmarkStore) SetBookmark(ctx context.Context, bm *storage.Bookmark) error {
	q := "INSERT INTO bookmarks (user_jid, room_jid, name, nick, password, autojoin) VALUES (" + b.s.phs(1, 6) + ") " +
		b.s.dialect.UpsertSuffix([]string{"user_jid", "room_jid"}, []string{"name", "nick", "password", "autojoin"})
	_, err := b.s.db.ExecContext(ctx, q, bm.UserJID, bm.RoomJID, bm.Name, bm.Nick, bm.Password, bm.Autojoin)
	return err
}

func (b *bookmarkStore) GetBookmark(ctx context.Context, userJID, roomJID string) (*storage.Bookmark, error) {
	var bm storage.Bookmark
	err := b.s.db.QueryRowContext(ctx,
		"SELECT user_jid, room_jid, name, nick, password, autojoin FROM bookmarks WHERE user_jid = "+b.s.ph(1)+" AND room_jid = "+b.s.ph(2),
		userJID, roomJID,
	).Scan(&bm.UserJID, &bm.RoomJID, &bm.Name, &bm.Nick, &bm.Password, &bm.Autojoin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &bm, nil
}

func (b *bookmarkStore) GetBookmarks(ctx context.Context, userJID string) ([]*storage.Bookmark, error) {
	rows, err := b.s.db.QueryContext(ctx,
		"SELECT user_jid, room_jid, name, nick, password, autojoin FROM bookmarks WHERE user_jid = "+b.s.ph(1), userJID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bms []*storage.Bookmark
	for rows.Next() {
		var bm storage.Bookmark
		if err := rows.Scan(&bm.UserJID, &bm.RoomJID, &bm.Name, &bm.Nick, &bm.Password, &bm.Autojoin); err != nil {
			return nil, err
		}
		bms = append(bms, &bm)
	}
	return bms, rows.Err()
}

func (b *bookmarkStore) DeleteBookmark(ctx context.Context, userJID, roomJID string) error {
	res, err := b.s.db.ExecContext(ctx,
		"DELETE FROM bookmarks WHERE user_jid = "+b.s.ph(1)+" AND room_jid = "+b.s.ph(2),
		userJID, roomJID,
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
