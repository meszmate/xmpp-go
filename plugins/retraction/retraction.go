// Package retraction implements XEP-0424 Message Retraction.
package retraction

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "retraction"

type Retract struct {
	XMLName xml.Name `xml:"urn:xmpp:message-retract:1 retract"`
	ID      string   `xml:"id,attr"`
}

type Retracted struct {
	XMLName xml.Name `xml:"urn:xmpp:message-retract:1 retracted"`
	Stamp   string   `xml:"stamp,attr,omitempty"`
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

func init() { _ = ns.Retraction }
