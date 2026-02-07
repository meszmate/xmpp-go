package storage

import (
	"context"
	"time"
)

// PubSubNode represents a publish-subscribe node.
type PubSubNode struct {
	Host    string // service JID
	NodeID  string
	Name    string
	Type    string // "leaf" or "collection"
	Config  map[string]string
	Creator string
}

// PubSubItem represents an item published to a node.
type PubSubItem struct {
	Host      string
	NodeID    string
	ItemID    string
	Publisher string
	Payload   []byte // raw XML
	CreatedAt time.Time
}

// PubSubSubscription represents a subscription to a node.
type PubSubSubscription struct {
	Host   string
	NodeID string
	JID    string
	SubID  string
	State  string // "subscribed", "pending", "unconfigured", "none"
}

// PubSubStore manages publish-subscribe data.
type PubSubStore interface {
	// CreateNode creates a new pubsub node.
	CreateNode(ctx context.Context, node *PubSubNode) error

	// GetNode retrieves a pubsub node.
	GetNode(ctx context.Context, host, nodeID string) (*PubSubNode, error)

	// DeleteNode deletes a pubsub node and all its items/subscriptions.
	DeleteNode(ctx context.Context, host, nodeID string) error

	// ListNodes lists all nodes for a host.
	ListNodes(ctx context.Context, host string) ([]*PubSubNode, error)

	// UpsertItem publishes or updates an item on a node.
	UpsertItem(ctx context.Context, item *PubSubItem) error

	// GetItem retrieves a specific item from a node.
	GetItem(ctx context.Context, host, nodeID, itemID string) (*PubSubItem, error)

	// GetItems retrieves all items from a node.
	GetItems(ctx context.Context, host, nodeID string) ([]*PubSubItem, error)

	// DeleteItem deletes an item from a node.
	DeleteItem(ctx context.Context, host, nodeID, itemID string) error

	// Subscribe adds a subscription to a node.
	Subscribe(ctx context.Context, sub *PubSubSubscription) error

	// Unsubscribe removes a subscription from a node.
	Unsubscribe(ctx context.Context, host, nodeID, jid string) error

	// GetSubscription retrieves a subscription.
	GetSubscription(ctx context.Context, host, nodeID, jid string) (*PubSubSubscription, error)

	// GetSubscriptions retrieves all subscriptions for a node.
	GetSubscriptions(ctx context.Context, host, nodeID string) ([]*PubSubSubscription, error)

	// GetUserSubscriptions retrieves all subscriptions for a user across all nodes.
	GetUserSubscriptions(ctx context.Context, host, jid string) ([]*PubSubSubscription, error)
}
