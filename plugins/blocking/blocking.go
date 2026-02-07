// Package blocking implements XEP-0191 Blocking Command.
package blocking

import (
	"context"
	"encoding/xml"
	"sync"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/storage"
)

const Name = "blocking"

type BlockList struct {
	XMLName xml.Name    `xml:"urn:xmpp:blocking blocklist"`
	Items   []BlockItem `xml:"item"`
}

type Block struct {
	XMLName xml.Name    `xml:"urn:xmpp:blocking block"`
	Items   []BlockItem `xml:"item"`
}

type Unblock struct {
	XMLName xml.Name    `xml:"urn:xmpp:blocking unblock"`
	Items   []BlockItem `xml:"item"`
}

type BlockItem struct {
	XMLName xml.Name `xml:"item"`
	JID     string   `xml:"jid,attr"`
}

type Plugin struct {
	mu      sync.RWMutex
	blocked map[string]bool // in-memory fallback
	store   storage.BlockingStore
	params  plugin.InitParams
}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }
func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	if params.Storage != nil {
		p.store = params.Storage.BlockingStore()
	}
	if p.store == nil {
		p.blocked = make(map[string]bool)
	}
	return nil
}
func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

func (p *Plugin) IsBlocked(ctx context.Context, jid string) (bool, error) {
	if p.store != nil {
		return p.store.IsBlocked(ctx, p.params.LocalJID(), jid)
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.blocked[jid], nil
}

func (p *Plugin) BlockJID(ctx context.Context, jid string) error {
	if p.store != nil {
		return p.store.BlockJID(ctx, p.params.LocalJID(), jid)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blocked[jid] = true
	return nil
}

func (p *Plugin) UnblockJID(ctx context.Context, jid string) error {
	if p.store != nil {
		return p.store.UnblockJID(ctx, p.params.LocalJID(), jid)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.blocked, jid)
	return nil
}

func (p *Plugin) BlockedList(ctx context.Context) ([]string, error) {
	if p.store != nil {
		return p.store.GetBlockedJIDs(ctx, p.params.LocalJID())
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	list := make([]string, 0, len(p.blocked))
	for jid := range p.blocked {
		list = append(list, jid)
	}
	return list, nil
}

func init() { _ = ns.Blocking }
