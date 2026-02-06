// Package forward implements XEP-0297 Stanza Forwarding.
package forward

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "forward"

// Forwarded wraps a forwarded stanza with optional delay.
type Forwarded struct {
	XMLName xml.Name `xml:"urn:xmpp:forward:0 forwarded"`
	Delay   *Delay   `xml:"urn:xmpp:delay delay,omitempty"`
	Inner   []byte   `xml:",innerxml"`
}

// Delay is an inline delay element for forwarded stanzas.
type Delay struct {
	XMLName xml.Name `xml:"urn:xmpp:delay delay"`
	From    string   `xml:"from,attr,omitempty"`
	Stamp   string   `xml:"stamp,attr"`
}

// Plugin implements XEP-0297.
type Plugin struct {
	params plugin.InitParams
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }
func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	return nil
}
func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

func init() { _ = ns.Forward }
