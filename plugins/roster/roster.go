// Package roster implements RFC 6121 Roster Management.
package roster

import (
	"context"
	"encoding/xml"
	"sync"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/storage"
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
	items  map[string]Item // in-memory fallback
	ver    string
	store  storage.RosterStore
	params plugin.InitParams
}

// New creates a new roster plugin.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }

func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	if params.Storage != nil {
		p.store = params.Storage.RosterStore()
	}
	if p.store == nil {
		p.items = make(map[string]Item)
	}
	// Handle server-initiated roster pushes (RFC 6121 §2.1.6).
	if params.Handle != nil {
		params.Handle(xml.Name{Space: ns.Roster, Local: "query"}, "", p.handlePush)
	}
	return nil
}

// handlePush applies a server roster push and acknowledges it.
func (p *Plugin) handlePush(ctx context.Context, st stanza.Stanza) error {
	iq, ok := st.(*stanza.IQ)
	if !ok || iq.Type != stanza.IQSet {
		return nil
	}
	var q Query
	if err := xml.Unmarshal(iq.Query, &q); err != nil {
		return nil
	}
	for _, it := range q.Items {
		if it.Subscription == SubRemove {
			_ = p.Remove(ctx, it.JID)
			continue
		}
		_ = p.Set(ctx, it)
	}
	if p.params.SendElement != nil {
		res := stanza.IQ{Header: stanza.Header{ID: iq.ID, Type: stanza.IQResult, To: iq.From}}
		return p.params.SendElement(ctx, &stanza.IQPayload{IQ: res})
	}
	return nil
}

// Fetch retrieves the roster from the server (RFC 6121 §2.1.3) and caches it
// locally. It requires a host that supplies InitParams.Request (an XMPP client);
// otherwise it returns the locally-stored items.
func (p *Plugin) Fetch(ctx context.Context) ([]Item, error) {
	if p.params.Request == nil {
		return p.Items(ctx)
	}
	req := stanza.NewIQ(stanza.IQGet)
	req.Query = []byte(`<query xmlns='jabber:iq:roster'/>`)
	res, err := p.params.Request(ctx, req)
	if err != nil {
		return nil, err
	}
	var q Query
	if err := xml.Unmarshal(res.Query, &q); err != nil {
		return nil, err
	}
	for _, it := range q.Items {
		_ = p.Set(ctx, it)
	}
	return q.Items, nil
}

func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

// Set adds or updates a roster item.
func (p *Plugin) Set(ctx context.Context, item Item) error {
	if p.store != nil {
		return p.store.UpsertRosterItem(ctx, &storage.RosterItem{
			UserJID:      p.params.LocalJID(),
			ContactJID:   item.JID,
			Name:         item.Name,
			Subscription: item.Subscription,
			Ask:          item.Ask,
			Groups:       item.Groups,
		})
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items[item.JID] = item
	return nil
}

// Remove removes a roster item.
func (p *Plugin) Remove(ctx context.Context, jid string) error {
	if p.store != nil {
		return p.store.DeleteRosterItem(ctx, p.params.LocalJID(), jid)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.items, jid)
	return nil
}

// Get returns a roster item by JID.
func (p *Plugin) Get(ctx context.Context, jid string) (Item, bool, error) {
	if p.store != nil {
		ri, err := p.store.GetRosterItem(ctx, p.params.LocalJID(), jid)
		if err != nil {
			if err == storage.ErrNotFound {
				return Item{}, false, nil
			}
			return Item{}, false, err
		}
		return rosterItemToItem(ri), true, nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	item, ok := p.items[jid]
	return item, ok, nil
}

// Items returns all roster items.
func (p *Plugin) Items(ctx context.Context) ([]Item, error) {
	if p.store != nil {
		ris, err := p.store.GetRosterItems(ctx, p.params.LocalJID())
		if err != nil {
			return nil, err
		}
		items := make([]Item, len(ris))
		for i, ri := range ris {
			items[i] = rosterItemToItem(ri)
		}
		return items, nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	items := make([]Item, 0, len(p.items))
	for _, item := range p.items {
		items = append(items, item)
	}
	return items, nil
}

// SetVersion sets the roster version.
func (p *Plugin) SetVersion(ctx context.Context, ver string) error {
	if p.store != nil {
		return p.store.SetRosterVersion(ctx, p.params.LocalJID(), ver)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ver = ver
	return nil
}

func rosterItemToItem(ri *storage.RosterItem) Item {
	return Item{
		JID:          ri.ContactJID,
		Name:         ri.Name,
		Subscription: ri.Subscription,
		Ask:          ri.Ask,
		Groups:       ri.Groups,
	}
}

func init() {
	_ = ns.Roster
}
