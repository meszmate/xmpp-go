// Package moderation implements XEP-0425 Message Moderation.
package moderation

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "moderation"

type Moderate struct {
	XMLName xml.Name `xml:"urn:xmpp:message-moderate:1 moderate"`
	ID      string   `xml:"id,attr"`
	Retract *Retract `xml:"retract,omitempty"`
	Reason  string   `xml:"reason,omitempty"`
}

type Retract struct {
	XMLName xml.Name `xml:"urn:xmpp:message-retract:1 retract"`
}

type Moderated struct {
	XMLName xml.Name `xml:"urn:xmpp:message-moderate:1 moderated"`
	By      string   `xml:"by,attr"`
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

func init() { _ = ns.Moderation }
