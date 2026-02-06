// Package bob implements XEP-0231 Bits of Binary.
package bob

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "bob"

type Data struct {
	XMLName xml.Name `xml:"urn:xmpp:bob data"`
	CID     string   `xml:"cid,attr"`
	Type    string   `xml:"type,attr"`
	MaxAge  int      `xml:"max-age,attr,omitempty"`
	Value   string   `xml:",chardata"`
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

func init() { _ = ns.BoB }
