// Package presence implements RFC 6121 Presence Management.
package presence

import (
	"context"
	"encoding/xml"
	"sync"

	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "presence"

// Status represents a presence status.
type Status struct {
	XMLName  xml.Name `xml:"presence"`
	Show     string   `xml:"show,omitempty"`
	Status   string   `xml:"status,omitempty"`
	Priority int8     `xml:"priority,omitempty"`
}

// Plugin implements presence management.
type Plugin struct {
	mu       sync.RWMutex
	roster   map[string]Status // jid -> last known status
	own      Status
	params   plugin.InitParams
}

// New creates a new presence plugin.
func New() *Plugin {
	return &Plugin{
		roster: make(map[string]Status),
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

// SetOwn updates our own presence status.
func (p *Plugin) SetOwn(status Status) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.own = status
}

// Own returns our current presence.
func (p *Plugin) Own() Status {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.own
}

// Update records a contact's presence.
func (p *Plugin) Update(jid string, status Status) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.roster[jid] = status
}

// Remove removes presence info for a contact.
func (p *Plugin) Remove(jid string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.roster, jid)
}

// GetPresence returns a contact's last known presence.
func (p *Plugin) GetPresence(jid string) (Status, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	s, ok := p.roster[jid]
	return s, ok
}

// Online returns all JIDs with known presence.
func (p *Plugin) Online() map[string]Status {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make(map[string]Status, len(p.roster))
	for k, v := range p.roster {
		result[k] = v
	}
	return result
}
