package sql

import (
	"context"
	"time"

	"github.com/meszmate/xmpp-go/storage"
)

type offlineStore struct{ s *Store }

func (o *offlineStore) StoreOfflineMessage(ctx context.Context, msg *storage.OfflineMessage) error {
	createdAt := msg.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	_, err := o.s.db.ExecContext(ctx,
		"INSERT INTO offline_messages (id, user_jid, from_jid, data, created_at) VALUES ("+o.s.phs(1, 5)+")",
		msg.ID, msg.UserJID, msg.FromJID, msg.Data, createdAt,
	)
	return err
}

func (o *offlineStore) GetOfflineMessages(ctx context.Context, userJID string) ([]*storage.OfflineMessage, error) {
	rows, err := o.s.db.QueryContext(ctx,
		"SELECT id, user_jid, from_jid, data, created_at FROM offline_messages WHERE user_jid = "+o.s.ph(1)+" ORDER BY created_at ASC",
		userJID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*storage.OfflineMessage
	for rows.Next() {
		var msg storage.OfflineMessage
		if err := rows.Scan(&msg.ID, &msg.UserJID, &msg.FromJID, &msg.Data, &msg.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, &msg)
	}
	return msgs, rows.Err()
}

func (o *offlineStore) DeleteOfflineMessages(ctx context.Context, userJID string) error {
	_, err := o.s.db.ExecContext(ctx, "DELETE FROM offline_messages WHERE user_jid = "+o.s.ph(1), userJID)
	return err
}

func (o *offlineStore) CountOfflineMessages(ctx context.Context, userJID string) (int, error) {
	var count int
	err := o.s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM offline_messages WHERE user_jid = "+o.s.ph(1), userJID,
	).Scan(&count)
	return count, err
}
