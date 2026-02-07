package sql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/meszmate/xmpp-go/storage"
)

type rosterStore struct{ s *Store }

func (r *rosterStore) UpsertRosterItem(ctx context.Context, item *storage.RosterItem) error {
	groups := strings.Join(item.Groups, "\n")
	q := "INSERT INTO roster_items (user_jid, contact_jid, name, subscription, ask, groups_list) VALUES (" + r.s.phs(1, 6) + ") " +
		r.s.dialect.UpsertSuffix([]string{"user_jid", "contact_jid"}, []string{"name", "subscription", "ask", "groups_list"})
	_, err := r.s.db.ExecContext(ctx, q, item.UserJID, item.ContactJID, item.Name, item.Subscription, item.Ask, groups)
	return err
}

func (r *rosterStore) GetRosterItem(ctx context.Context, userJID, contactJID string) (*storage.RosterItem, error) {
	row := r.s.db.QueryRowContext(ctx,
		"SELECT user_jid, contact_jid, name, subscription, ask, groups_list FROM roster_items WHERE user_jid = "+r.s.ph(1)+" AND contact_jid = "+r.s.ph(2),
		userJID, contactJID,
	)
	return scanRosterItem(row)
}

func (r *rosterStore) GetRosterItems(ctx context.Context, userJID string) ([]*storage.RosterItem, error) {
	rows, err := r.s.db.QueryContext(ctx,
		"SELECT user_jid, contact_jid, name, subscription, ask, groups_list FROM roster_items WHERE user_jid = "+r.s.ph(1),
		userJID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*storage.RosterItem
	for rows.Next() {
		var item storage.RosterItem
		var groups string
		if err := rows.Scan(&item.UserJID, &item.ContactJID, &item.Name, &item.Subscription, &item.Ask, &groups); err != nil {
			return nil, err
		}
		if groups != "" {
			item.Groups = strings.Split(groups, "\n")
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}

func (r *rosterStore) DeleteRosterItem(ctx context.Context, userJID, contactJID string) error {
	res, err := r.s.db.ExecContext(ctx,
		"DELETE FROM roster_items WHERE user_jid = "+r.s.ph(1)+" AND contact_jid = "+r.s.ph(2),
		userJID, contactJID,
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

func (r *rosterStore) GetRosterVersion(ctx context.Context, userJID string) (string, error) {
	var ver string
	err := r.s.db.QueryRowContext(ctx,
		"SELECT version FROM roster_versions WHERE user_jid = "+r.s.ph(1), userJID,
	).Scan(&ver)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return ver, nil
}

func (r *rosterStore) SetRosterVersion(ctx context.Context, userJID, version string) error {
	q := "INSERT INTO roster_versions (user_jid, version) VALUES (" + r.s.phs(1, 2) + ") " +
		r.s.dialect.UpsertSuffix([]string{"user_jid"}, []string{"version"})
	_, err := r.s.db.ExecContext(ctx, q, userJID, version)
	return err
}

func scanRosterItem(row *sql.Row) (*storage.RosterItem, error) {
	var item storage.RosterItem
	var groups string
	err := row.Scan(&item.UserJID, &item.ContactJID, &item.Name, &item.Subscription, &item.Ask, &groups)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if groups != "" {
		item.Groups = strings.Split(groups, "\n")
	}
	return &item, nil
}
