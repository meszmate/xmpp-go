package sql

import (
	"context"
	"fmt"
	"time"

	"github.com/meszmate/xmpp-go/storage"
)

type mamStore struct{ s *Store }

func (m *mamStore) ArchiveMessage(ctx context.Context, msg *storage.ArchivedMessage) error {
	createdAt := msg.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	_, err := m.s.db.ExecContext(ctx,
		"INSERT INTO mam_messages (id, user_jid, with_jid, from_jid, data, created_at) VALUES ("+m.s.phs(1, 6)+")",
		msg.ID, msg.UserJID, msg.WithJID, msg.FromJID, msg.Data, createdAt,
	)
	return err
}

func (m *mamStore) QueryMessages(ctx context.Context, query *storage.MAMQuery) (*storage.MAMResult, error) {
	where := "WHERE user_jid = " + m.s.ph(1)
	args := []any{query.UserJID}
	n := 2

	if query.WithJID != "" {
		where += " AND with_jid = " + m.s.ph(n)
		args = append(args, query.WithJID)
		n++
	}
	if !query.Start.IsZero() {
		where += " AND created_at >= " + m.s.ph(n)
		args = append(args, query.Start)
		n++
	}
	if !query.End.IsZero() {
		where += " AND created_at <= " + m.s.ph(n)
		args = append(args, query.End)
		n++
	}
	if query.AfterID != "" {
		where += " AND id > " + m.s.ph(n)
		args = append(args, query.AfterID)
		n++
	}
	if query.BeforeID != "" {
		where += " AND id < " + m.s.ph(n)
		args = append(args, query.BeforeID)
		n++
	}

	max := query.Max
	if max <= 0 {
		max = 100
	}

	q := fmt.Sprintf("SELECT id, user_jid, with_jid, from_jid, data, created_at FROM mam_messages %s ORDER BY created_at ASC LIMIT %d", where, max+1)
	rows, err := m.s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*storage.ArchivedMessage
	for rows.Next() {
		var msg storage.ArchivedMessage
		if err := rows.Scan(&msg.ID, &msg.UserJID, &msg.WithJID, &msg.FromJID, &msg.Data, &msg.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, &msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	complete := len(msgs) <= max
	if len(msgs) > max {
		msgs = msgs[:max]
	}

	result := &storage.MAMResult{
		Messages: msgs,
		Complete: complete,
		Count:    len(msgs),
	}
	if len(msgs) > 0 {
		result.First = msgs[0].ID
		result.Last = msgs[len(msgs)-1].ID
	}
	return result, nil
}

func (m *mamStore) DeleteMessageArchive(ctx context.Context, userJID string) error {
	_, err := m.s.db.ExecContext(ctx, "DELETE FROM mam_messages WHERE user_jid = "+m.s.ph(1), userJID)
	return err
}
