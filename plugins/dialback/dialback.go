// Package dialback implements XEP-0220 Server Dialback and XEP-0288 Bidirectional S2S.
package dialback

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "dialback"

type Result struct {
	XMLName xml.Name `xml:"jabber:server:dialback result"`
	From    string   `xml:"from,attr"`
	To      string   `xml:"to,attr"`
	Type    string   `xml:"type,attr,omitempty"`
	Key     string   `xml:",chardata"`
}

type Verify struct {
	XMLName xml.Name `xml:"jabber:server:dialback verify"`
	From    string   `xml:"from,attr"`
	To      string   `xml:"to,attr"`
	ID      string   `xml:"id,attr"`
	Type    string   `xml:"type,attr,omitempty"`
	Key     string   `xml:",chardata"`
}

// Bidi represents XEP-0288 bidirectional S2S.
type Bidi struct {
	XMLName xml.Name `xml:"urn:xmpp:bidi bidi"`
}

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

func init() {
	_ = ns.Dialback
	_ = ns.BidiS2S
}
