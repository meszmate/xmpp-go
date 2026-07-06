// Package ping implements XEP-0199 XMPP Ping.
package ping

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/stanza"
)

const Name = "ping"

// Ping represents an XMPP ping element.
type Ping struct {
	XMLName xml.Name `xml:"urn:xmpp:ping ping"`
}

// Plugin implements XEP-0199.
type Plugin struct {
	params plugin.InitParams
}

// New creates a new ping plugin.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }

func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	if params.Handle != nil {
		params.Handle(xml.Name{Space: ns.Ping, Local: "ping"}, "", p.handle)
	}
	return nil
}

// handle answers an inbound XEP-0199 ping with a pong.
func (p *Plugin) handle(ctx context.Context, st stanza.Stanza) error {
	iq, ok := st.(*stanza.IQ)
	if !ok || iq.Type != stanza.IQGet {
		return nil
	}
	res := stanza.IQ{Header: stanza.Header{ID: iq.ID, Type: stanza.IQResult, To: iq.From}}
	return p.params.SendElement(ctx, &stanza.IQPayload{IQ: res})
}

// Ping sends a XEP-0199 ping to the target JID and waits for the pong. It
// requires a host that supplies InitParams.Request (an XMPP client).
func (p *Plugin) Ping(ctx context.Context, to jid.JID) error {
	if p.params.Request == nil {
		return nil
	}
	req := stanza.NewIQ(stanza.IQGet)
	req.To = to
	req.Query = []byte(`<ping xmlns='urn:xmpp:ping'/>`)
	_, err := p.params.Request(ctx, req)
	return err
}

func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

func init() {
	_ = ns.Ping
}
