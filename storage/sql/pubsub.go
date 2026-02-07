package sql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/meszmate/xmpp-go/storage"
)

type pubsubStore struct{ s *Store }

func (p *pubsubStore) CreateNode(ctx context.Context, node *storage.PubSubNode) error {
	_, err := p.s.db.ExecContext(ctx,
		"INSERT INTO pubsub_nodes (host, node_id, name, type, creator) VALUES ("+p.s.phs(1, 5)+")",
		node.Host, node.NodeID, node.Name, node.Type, node.Creator,
	)
	if err != nil && isUniqueViolation(err) {
		return storage.ErrUserExists
	}
	return err
}

func (p *pubsubStore) GetNode(ctx context.Context, host, nodeID string) (*storage.PubSubNode, error) {
	var node storage.PubSubNode
	err := p.s.db.QueryRowContext(ctx,
		"SELECT host, node_id, name, type, creator FROM pubsub_nodes WHERE host = "+p.s.ph(1)+" AND node_id = "+p.s.ph(2),
		host, nodeID,
	).Scan(&node.Host, &node.NodeID, &node.Name, &node.Type, &node.Creator)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (p *pubsubStore) DeleteNode(ctx context.Context, host, nodeID string) error {
	res, err := p.s.db.ExecContext(ctx,
		"DELETE FROM pubsub_nodes WHERE host = "+p.s.ph(1)+" AND node_id = "+p.s.ph(2),
		host, nodeID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return storage.ErrNotFound
	}
	_, _ = p.s.db.ExecContext(ctx, "DELETE FROM pubsub_items WHERE host = "+p.s.ph(1)+" AND node_id = "+p.s.ph(2), host, nodeID)
	_, _ = p.s.db.ExecContext(ctx, "DELETE FROM pubsub_subscriptions WHERE host = "+p.s.ph(1)+" AND node_id = "+p.s.ph(2), host, nodeID)
	return nil
}

func (p *pubsubStore) ListNodes(ctx context.Context, host string) ([]*storage.PubSubNode, error) {
	rows, err := p.s.db.QueryContext(ctx,
		"SELECT host, node_id, name, type, creator FROM pubsub_nodes WHERE host = "+p.s.ph(1), host,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*storage.PubSubNode
	for rows.Next() {
		var node storage.PubSubNode
		if err := rows.Scan(&node.Host, &node.NodeID, &node.Name, &node.Type, &node.Creator); err != nil {
			return nil, err
		}
		nodes = append(nodes, &node)
	}
	return nodes, rows.Err()
}

func (p *pubsubStore) UpsertItem(ctx context.Context, item *storage.PubSubItem) error {
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	q := "INSERT INTO pubsub_items (host, node_id, item_id, publisher, payload, created_at) VALUES (" + p.s.phs(1, 6) + ") " +
		p.s.dialect.UpsertSuffix([]string{"host", "node_id", "item_id"}, []string{"publisher", "payload", "created_at"})
	_, err := p.s.db.ExecContext(ctx, q, item.Host, item.NodeID, item.ItemID, item.Publisher, item.Payload, createdAt)
	return err
}

func (p *pubsubStore) GetItem(ctx context.Context, host, nodeID, itemID string) (*storage.PubSubItem, error) {
	var item storage.PubSubItem
	err := p.s.db.QueryRowContext(ctx,
		"SELECT host, node_id, item_id, publisher, payload, created_at FROM pubsub_items WHERE host = "+p.s.ph(1)+" AND node_id = "+p.s.ph(2)+" AND item_id = "+p.s.ph(3),
		host, nodeID, itemID,
	).Scan(&item.Host, &item.NodeID, &item.ItemID, &item.Publisher, &item.Payload, &item.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (p *pubsubStore) GetItems(ctx context.Context, host, nodeID string) ([]*storage.PubSubItem, error) {
	rows, err := p.s.db.QueryContext(ctx,
		"SELECT host, node_id, item_id, publisher, payload, created_at FROM pubsub_items WHERE host = "+p.s.ph(1)+" AND node_id = "+p.s.ph(2)+" ORDER BY created_at ASC",
		host, nodeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*storage.PubSubItem
	for rows.Next() {
		var item storage.PubSubItem
		if err := rows.Scan(&item.Host, &item.NodeID, &item.ItemID, &item.Publisher, &item.Payload, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}

func (p *pubsubStore) DeleteItem(ctx context.Context, host, nodeID, itemID string) error {
	res, err := p.s.db.ExecContext(ctx,
		"DELETE FROM pubsub_items WHERE host = "+p.s.ph(1)+" AND node_id = "+p.s.ph(2)+" AND item_id = "+p.s.ph(3),
		host, nodeID, itemID,
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

func (p *pubsubStore) Subscribe(ctx context.Context, sub *storage.PubSubSubscription) error {
	q := "INSERT INTO pubsub_subscriptions (host, node_id, jid, sub_id, state) VALUES (" + p.s.phs(1, 5) + ") " +
		p.s.dialect.UpsertSuffix([]string{"host", "node_id", "jid"}, []string{"sub_id", "state"})
	_, err := p.s.db.ExecContext(ctx, q, sub.Host, sub.NodeID, sub.JID, sub.SubID, sub.State)
	return err
}

func (p *pubsubStore) Unsubscribe(ctx context.Context, host, nodeID, jid string) error {
	_, err := p.s.db.ExecContext(ctx,
		"DELETE FROM pubsub_subscriptions WHERE host = "+p.s.ph(1)+" AND node_id = "+p.s.ph(2)+" AND jid = "+p.s.ph(3),
		host, nodeID, jid,
	)
	return err
}

func (p *pubsubStore) GetSubscription(ctx context.Context, host, nodeID, jid string) (*storage.PubSubSubscription, error) {
	var sub storage.PubSubSubscription
	err := p.s.db.QueryRowContext(ctx,
		"SELECT host, node_id, jid, sub_id, state FROM pubsub_subscriptions WHERE host = "+p.s.ph(1)+" AND node_id = "+p.s.ph(2)+" AND jid = "+p.s.ph(3),
		host, nodeID, jid,
	).Scan(&sub.Host, &sub.NodeID, &sub.JID, &sub.SubID, &sub.State)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (p *pubsubStore) GetSubscriptions(ctx context.Context, host, nodeID string) ([]*storage.PubSubSubscription, error) {
	rows, err := p.s.db.QueryContext(ctx,
		"SELECT host, node_id, jid, sub_id, state FROM pubsub_subscriptions WHERE host = "+p.s.ph(1)+" AND node_id = "+p.s.ph(2),
		host, nodeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*storage.PubSubSubscription
	for rows.Next() {
		var sub storage.PubSubSubscription
		if err := rows.Scan(&sub.Host, &sub.NodeID, &sub.JID, &sub.SubID, &sub.State); err != nil {
			return nil, err
		}
		subs = append(subs, &sub)
	}
	return subs, rows.Err()
}

func (p *pubsubStore) GetUserSubscriptions(ctx context.Context, host, jid string) ([]*storage.PubSubSubscription, error) {
	rows, err := p.s.db.QueryContext(ctx,
		"SELECT host, node_id, jid, sub_id, state FROM pubsub_subscriptions WHERE host = "+p.s.ph(1)+" AND jid = "+p.s.ph(2),
		host, jid,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*storage.PubSubSubscription
	for rows.Next() {
		var sub storage.PubSubSubscription
		if err := rows.Scan(&sub.Host, &sub.NodeID, &sub.JID, &sub.SubID, &sub.State); err != nil {
			return nil, err
		}
		subs = append(subs, &sub)
	}
	return subs, rows.Err()
}
