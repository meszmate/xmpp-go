// Package stanzaid implements XEP-0359 Unique and Stable Stanza IDs.
package stanzaid

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "stanzaid"

// StanzaID represents a stanza-id element.
type StanzaID struct {
	XMLName xml.Name `xml:"urn:xmpp:sid:0 stanza-id"`
	ID      string   `xml:"id,attr"`
	By      string   `xml:"by,attr"`
}

// OriginID represents an origin-id element.
type OriginID struct {
	XMLName xml.Name `xml:"urn:xmpp:sid:0 origin-id"`
	ID      string   `xml:"id,attr"`
}

// Plugin implements XEP-0359.
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

func init() { _ = ns.StanzaID }
