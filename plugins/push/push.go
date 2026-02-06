// Package push implements XEP-0357 Push Notifications.
package push

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "push"

type Enable struct {
	XMLName xml.Name `xml:"urn:xmpp:push:0 enable"`
	JID     string   `xml:"jid,attr"`
	Node    string   `xml:"node,attr"`
	Form    []byte   `xml:",innerxml"`
}

type Disable struct {
	XMLName xml.Name `xml:"urn:xmpp:push:0 disable"`
	JID     string   `xml:"jid,attr"`
	Node    string   `xml:"node,attr,omitempty"`
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

func init() { _ = ns.Push }
