// Package pubsub implements XEP-0060 Publish-Subscribe and XEP-0163 PEP.
package pubsub

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/storage"
)

const Name = "pubsub"

type PubSub struct {
	XMLName     xml.Name     `xml:"http://jabber.org/protocol/pubsub pubsub"`
	Create      *Create      `xml:"create,omitempty"`
	Configure   *Configure   `xml:"configure,omitempty"`
	Subscribe   *SubReq      `xml:"subscribe,omitempty"`
	Unsubscribe *Unsub       `xml:"unsubscribe,omitempty"`
	Publish     *Publish     `xml:"publish,omitempty"`
	Retract     *Retract     `xml:"retract,omitempty"`
	Items       *Items       `xml:"items,omitempty"`
	Subscription *Subscription `xml:"subscription,omitempty"`
}

type Create struct {
	XMLName xml.Name `xml:"create"`
	Node    string   `xml:"node,attr,omitempty"`
}

type Configure struct {
	XMLName xml.Name `xml:"configure"`
	Form    []byte   `xml:",innerxml"`
}

type SubReq struct {
	XMLName xml.Name `xml:"subscribe"`
	Node    string   `xml:"node,attr"`
	JID     string   `xml:"jid,attr"`
}

type Unsub struct {
	XMLName xml.Name `xml:"unsubscribe"`
	Node    string   `xml:"node,attr"`
	JID     string   `xml:"jid,attr"`
	SubID   string   `xml:"subid,attr,omitempty"`
}

type Publish struct {
	XMLName xml.Name `xml:"publish"`
	Node    string   `xml:"node,attr"`
	Items   []PubItem `xml:"item"`
}

type PubItem struct {
	XMLName xml.Name `xml:"item"`
	ID      string   `xml:"id,attr,omitempty"`
	Payload []byte   `xml:",innerxml"`
}

type Retract struct {
	XMLName xml.Name `xml:"retract"`
	Node    string   `xml:"node,attr"`
	Notify  bool     `xml:"notify,attr,omitempty"`
	Items   []PubItem `xml:"item"`
}

type Items struct {
	XMLName xml.Name  `xml:"items"`
	Node    string    `xml:"node,attr"`
	SubID   string    `xml:"subid,attr,omitempty"`
	MaxItems *int     `xml:"max_items,attr,omitempty"`
	Items   []PubItem `xml:"item"`
}

type Subscription struct {
	XMLName xml.Name `xml:"subscription"`
	Node    string   `xml:"node,attr"`
	JID     string   `xml:"jid,attr"`
	SubID   string   `xml:"subid,attr,omitempty"`
	State   string   `xml:"subscription,attr,omitempty"`
}

// Event is from PubSub event notifications.
type Event struct {
	XMLName xml.Name     `xml:"http://jabber.org/protocol/pubsub#event event"`
	Items   *EventItems  `xml:"items,omitempty"`
	Purge   *EventPurge  `xml:"purge,omitempty"`
	Delete  *EventDelete `xml:"delete,omitempty"`
}

type EventItems struct {
	XMLName xml.Name  `xml:"items"`
	Node    string    `xml:"node,attr"`
	Items   []PubItem `xml:"item"`
	Retract []EventRetract `xml:"retract"`
}

type EventRetract struct {
	XMLName xml.Name `xml:"retract"`
	ID      string   `xml:"id,attr"`
}

type EventPurge struct {
	XMLName xml.Name `xml:"purge"`
	Node    string   `xml:"node,attr"`
}

type EventDelete struct {
	XMLName xml.Name `xml:"delete"`
	Node    string   `xml:"node,attr"`
}

// Owner types
type PubSubOwner struct {
	XMLName   xml.Name   `xml:"http://jabber.org/protocol/pubsub#owner pubsub"`
	Configure *OwnerConfigure `xml:"configure,omitempty"`
	Delete    *OwnerDelete    `xml:"delete,omitempty"`
	Purge     *OwnerPurge     `xml:"purge,omitempty"`
}

type OwnerConfigure struct {
	XMLName xml.Name `xml:"configure"`
	Node    string   `xml:"node,attr"`
	Form    []byte   `xml:",innerxml"`
}

type OwnerDelete struct {
	XMLName xml.Name `xml:"delete"`
	Node    string   `xml:"node,attr"`
}

type OwnerPurge struct {
	XMLName xml.Name `xml:"purge"`
	Node    string   `xml:"node,attr"`
}

type Plugin struct {
	store  storage.PubSubStore
	params plugin.InitParams
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }
func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	if params.Storage != nil {
		p.store = params.Storage.PubSubStore()
	}
	return nil
}
func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

// CreateNode creates a new pubsub node. Returns nil if no store is configured.
func (p *Plugin) CreateNode(ctx context.Context, node *storage.PubSubNode) error {
	if p.store == nil {
		return nil
	}
	return p.store.CreateNode(ctx, node)
}

// GetNode retrieves a pubsub node. Returns nil if no store is configured.
func (p *Plugin) GetNode(ctx context.Context, host, nodeID string) (*storage.PubSubNode, error) {
	if p.store == nil {
		return nil, nil
	}
	return p.store.GetNode(ctx, host, nodeID)
}

// DeleteNode deletes a pubsub node. Returns nil if no store is configured.
func (p *Plugin) DeleteNode(ctx context.Context, host, nodeID string) error {
	if p.store == nil {
		return nil
	}
	return p.store.DeleteNode(ctx, host, nodeID)
}

// ListNodes lists all nodes for a host. Returns nil if no store is configured.
func (p *Plugin) ListNodes(ctx context.Context, host string) ([]*storage.PubSubNode, error) {
	if p.store == nil {
		return nil, nil
	}
	return p.store.ListNodes(ctx, host)
}

// PublishItem publishes or updates an item on a node. Returns nil if no store is configured.
func (p *Plugin) PublishItem(ctx context.Context, item *storage.PubSubItem) error {
	if p.store == nil {
		return nil
	}
	return p.store.UpsertItem(ctx, item)
}

// GetItems retrieves all items from a node. Returns nil if no store is configured.
func (p *Plugin) GetItems(ctx context.Context, host, nodeID string) ([]*storage.PubSubItem, error) {
	if p.store == nil {
		return nil, nil
	}
	return p.store.GetItems(ctx, host, nodeID)
}

// DeleteItem deletes an item from a node. Returns nil if no store is configured.
func (p *Plugin) DeleteItem(ctx context.Context, host, nodeID, itemID string) error {
	if p.store == nil {
		return nil
	}
	return p.store.DeleteItem(ctx, host, nodeID, itemID)
}

// SubscribeNode adds a subscription. Returns nil if no store is configured.
func (p *Plugin) SubscribeNode(ctx context.Context, sub *storage.PubSubSubscription) error {
	if p.store == nil {
		return nil
	}
	return p.store.Subscribe(ctx, sub)
}

// UnsubscribeNode removes a subscription. Returns nil if no store is configured.
func (p *Plugin) UnsubscribeNode(ctx context.Context, host, nodeID, jid string) error {
	if p.store == nil {
		return nil
	}
	return p.store.Unsubscribe(ctx, host, nodeID, jid)
}

// GetSubscriptions retrieves all subscriptions for a node. Returns nil if no store is configured.
func (p *Plugin) GetSubscriptions(ctx context.Context, host, nodeID string) ([]*storage.PubSubSubscription, error) {
	if p.store == nil {
		return nil, nil
	}
	return p.store.GetSubscriptions(ctx, host, nodeID)
}

func init() {
	_ = ns.PubSub
	_ = ns.PubSubEvent
	_ = ns.PubSubOwner
}
