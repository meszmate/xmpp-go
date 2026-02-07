package sql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/meszmate/xmpp-go/storage"
)

type vcardStore struct{ s *Store }

func (v *vcardStore) SetVCard(ctx context.Context, userJID string, data []byte) error {
	q := "INSERT INTO vcards (user_jid, data) VALUES (" + v.s.phs(1, 2) + ") " +
		v.s.dialect.UpsertSuffix([]string{"user_jid"}, []string{"data"})
	_, err := v.s.db.ExecContext(ctx, q, userJID, data)
	return err
}

func (v *vcardStore) GetVCard(ctx context.Context, userJID string) ([]byte, error) {
	var data []byte
	err := v.s.db.QueryRowContext(ctx,
		"SELECT data FROM vcards WHERE user_jid = "+v.s.ph(1), userJID,
	).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (v *vcardStore) DeleteVCard(ctx context.Context, userJID string) error {
	res, err := v.s.db.ExecContext(ctx, "DELETE FROM vcards WHERE user_jid = "+v.s.ph(1), userJID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}
