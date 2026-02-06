// Package blocking implements XEP-0191 Blocking Command.
package blocking

import (
	"context"
	"encoding/xml"
	"sync"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
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
	blocked map[string]bool
	params  plugin.InitParams
}

func New() *Plugin {
	return &Plugin{blocked: make(map[string]bool)}
}

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }
func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	return nil
}
func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

func (p *Plugin) IsBlocked(jid string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.blocked[jid]
}

func (p *Plugin) BlockJID(jid string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blocked[jid] = true
}

func (p *Plugin) UnblockJID(jid string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.blocked, jid)
}

func (p *Plugin) BlockedList() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	list := make([]string, 0, len(p.blocked))
	for jid := range p.blocked {
		list = append(list, jid)
	}
	return list
}

func init() { _ = ns.Blocking }
