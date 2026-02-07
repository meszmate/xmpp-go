package sql

import (
	"context"
)

type blockingStore struct{ s *Store }

func (b *blockingStore) BlockJID(ctx context.Context, userJID, blockedJID string) error {
	q := "INSERT INTO blocked_jids (user_jid, blocked_jid) VALUES (" + b.s.phs(1, 2) + ") " +
		b.s.dialect.UpsertSuffix([]string{"user_jid", "blocked_jid"}, nil)
	_, err := b.s.db.ExecContext(ctx, q, userJID, blockedJID)
	return err
}

func (b *blockingStore) UnblockJID(ctx context.Context, userJID, blockedJID string) error {
	_, err := b.s.db.ExecContext(ctx,
		"DELETE FROM blocked_jids WHERE user_jid = "+b.s.ph(1)+" AND blocked_jid = "+b.s.ph(2),
		userJID, blockedJID,
	)
	return err
}

func (b *blockingStore) IsBlocked(ctx context.Context, userJID, blockedJID string) (bool, error) {
	var count int
	err := b.s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM blocked_jids WHERE user_jid = "+b.s.ph(1)+" AND blocked_jid = "+b.s.ph(2),
		userJID, blockedJID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (b *blockingStore) GetBlockedJIDs(ctx context.Context, userJID string) ([]string, error) {
	rows, err := b.s.db.QueryContext(ctx,
		"SELECT blocked_jid FROM blocked_jids WHERE user_jid = "+b.s.ph(1), userJID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jids []string
	for rows.Next() {
		var jid string
		if err := rows.Scan(&jid); err != nil {
			return nil, err
		}
		jids = append(jids, jid)
	}
	return jids, rows.Err()
}
