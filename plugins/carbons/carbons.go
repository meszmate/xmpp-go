// Package carbons implements XEP-0280 Message Carbons.
package carbons

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "carbons"

type Enable struct {
	XMLName xml.Name `xml:"urn:xmpp:carbons:2 enable"`
}

type Disable struct {
	XMLName xml.Name `xml:"urn:xmpp:carbons:2 disable"`
}

type Sent struct {
	XMLName   xml.Name `xml:"urn:xmpp:carbons:2 sent"`
	Forwarded []byte   `xml:",innerxml"`
}

type Received struct {
	XMLName   xml.Name `xml:"urn:xmpp:carbons:2 received"`
	Forwarded []byte   `xml:",innerxml"`
}

type Private struct {
	XMLName xml.Name `xml:"urn:xmpp:carbons:2 private"`
}

type Plugin struct {
	enabled bool
	params  plugin.InitParams
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

func (p *Plugin) IsEnabled() bool  { return p.enabled }
func (p *Plugin) SetEnabled(v bool) { p.enabled = v }

func init() { _ = ns.Carbons }
