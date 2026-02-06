// Package roster implements RFC 6121 Roster Management.
package roster

import (
	"context"
	"encoding/xml"
	"sync"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "roster"

// Subscription types.
const (
	SubNone   = "none"
	SubTo     = "to"
	SubFrom   = "from"
	SubBoth   = "both"
	SubRemove = "remove"
)

// Item represents a roster item.
type Item struct {
	XMLName      xml.Name `xml:"item"`
	JID          string   `xml:"jid,attr"`
	Name         string   `xml:"name,attr,omitempty"`
	Subscription string   `xml:"subscription,attr,omitempty"`
	Ask          string   `xml:"ask,attr,omitempty"`
	Groups       []string `xml:"group,omitempty"`
}

// Query represents a roster query.
type Query struct {
	XMLName xml.Name `xml:"jabber:iq:roster query"`
	Ver     string   `xml:"ver,attr,omitempty"`
	Items   []Item   `xml:"item"`
}

// Plugin implements roster management.
type Plugin struct {
	mu     sync.RWMutex
	items  map[string]Item
	ver    string
	params plugin.InitParams
}

// New creates a new roster plugin.
func New() *Plugin {
	return &Plugin{
		items: make(map[string]Item),
	}
}

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }

func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	return nil
}

func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

// Set adds or updates a roster item.
func (p *Plugin) Set(item Item) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items[item.JID] = item
}

// Remove removes a roster item.
func (p *Plugin) Remove(jid string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.items, jid)
}

// Get returns a roster item by JID.
func (p *Plugin) Get(jid string) (Item, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	item, ok := p.items[jid]
	return item, ok
}

// Items returns all roster items.
func (p *Plugin) Items() []Item {
	p.mu.RLock()
	defer p.mu.RUnlock()
	items := make([]Item, 0, len(p.items))
	for _, item := range p.items {
		items = append(items, item)
	}
	return items
}

// SetVersion sets the roster version.
func (p *Plugin) SetVersion(ver string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ver = ver
}

func init() {
	_ = ns.Roster
}
