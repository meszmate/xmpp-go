// Package lastactivity implements XEP-0012 Last Activity.
package lastactivity

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "lastactivity"

type Query struct {
	XMLName xml.Name `xml:"jabber:iq:last query"`
	Seconds uint64   `xml:"seconds,attr,omitempty"`
	Status  string   `xml:",chardata"`
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

func init() { _ = ns.LastActivity }
