// Package reactions implements XEP-0444 Message Reactions.
package reactions

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "reactions"

type Reactions struct {
	XMLName xml.Name   `xml:"urn:xmpp:reactions:0 reactions"`
	ID      string     `xml:"id,attr"`
	Items   []Reaction `xml:"reaction"`
}

type Reaction struct {
	XMLName xml.Name `xml:"reaction"`
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

func init() { _ = ns.Reactions }
