// Package ping implements XEP-0199 XMPP Ping.
package ping

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "ping"

// Ping represents an XMPP ping element.
type Ping struct {
	XMLName xml.Name `xml:"urn:xmpp:ping ping"`
}

// Plugin implements XEP-0199.
type Plugin struct {
	params plugin.InitParams
}

// New creates a new ping plugin.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }

func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	return nil
}

func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

func init() {
	_ = ns.Ping
}
