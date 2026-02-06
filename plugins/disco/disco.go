// Package disco implements XEP-0030 Service Discovery.
package disco

import (
	"context"
	"encoding/xml"
	"sync"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "disco"

// Identity represents a disco identity.
type Identity struct {
	XMLName  xml.Name `xml:"identity"`
	Category string   `xml:"category,attr"`
	Type     string   `xml:"type,attr"`
	Name     string   `xml:"name,attr,omitempty"`
	Lang     string   `xml:"xml:lang,attr,omitempty"`
}

// Feature represents a disco feature.
type Feature struct {
	XMLName xml.Name `xml:"feature"`
	Var     string   `xml:"var,attr"`
}

// InfoQuery represents a disco#info query.
type InfoQuery struct {
	XMLName    xml.Name   `xml:"http://jabber.org/protocol/disco#info query"`
	Node       string     `xml:"node,attr,omitempty"`
	Identities []Identity `xml:"identity"`
	Features   []Feature  `xml:"feature"`
}

// Item represents a disco item.
type Item struct {
	XMLName xml.Name `xml:"item"`
	JID     string   `xml:"jid,attr"`
	Node    string   `xml:"node,attr,omitempty"`
	Name    string   `xml:"name,attr,omitempty"`
}

// ItemsQuery represents a disco#items query.
type ItemsQuery struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/disco#items query"`
	Node    string   `xml:"node,attr,omitempty"`
	Items   []Item   `xml:"item"`
}

// Plugin implements XEP-0030 Service Discovery.
type Plugin struct {
	mu         sync.RWMutex
	identities []Identity
	features   []Feature
	items      []Item
	params     plugin.InitParams
}

// New creates a new disco plugin.
func New() *Plugin {
	return &Plugin{
		features: []Feature{
			{Var: ns.DiscoInfo},
			{Var: ns.DiscoItems},
		},
	}
}

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }

func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	return nil
}

func (p *Plugin) Close() error              { return nil }
func (p *Plugin) Dependencies() []string    { return nil }

// AddIdentity adds an identity to the disco response.
func (p *Plugin) AddIdentity(identity Identity) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.identities = append(p.identities, identity)
}

// AddFeature adds a feature to the disco response.
func (p *Plugin) AddFeature(feature string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.features = append(p.features, Feature{Var: feature})
}

// AddItem adds an item to the disco response.
func (p *Plugin) AddItem(item Item) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items = append(p.items, item)
}

// Info returns the service discovery info.
func (p *Plugin) Info() InfoQuery {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return InfoQuery{
		Identities: append([]Identity(nil), p.identities...),
		Features:   append([]Feature(nil), p.features...),
	}
}

// Items returns the service discovery items.
func (p *Plugin) Items() ItemsQuery {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return ItemsQuery{
		Items: append([]Item(nil), p.items...),
	}
}
