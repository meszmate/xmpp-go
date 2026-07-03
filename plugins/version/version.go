// Package version implements XEP-0092 Software Version.
package version

import (
	"context"
	"encoding/xml"
	"runtime"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/stanza"
)

const Name = "version"

// Query represents a software version query.
type Query struct {
	XMLName xml.Name `xml:"jabber:iq:version query"`
	Name    string   `xml:"name,omitempty"`
	Version string   `xml:"version,omitempty"`
	OS      string   `xml:"os,omitempty"`
}

// Plugin implements XEP-0092.
type Plugin struct {
	info   Query
	params plugin.InitParams
}

// New creates a new version plugin.
func New(name, version string) *Plugin {
	return &Plugin{
		info: Query{
			Name:    name,
			Version: version,
			OS:      runtime.GOOS,
		},
	}
}

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }

func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	if params.Handle != nil {
		params.Handle(xml.Name{Space: ns.Version, Local: "query"}, "", p.handle)
	}
	return nil
}

// handle answers a XEP-0092 software version query.
func (p *Plugin) handle(ctx context.Context, st stanza.Stanza) error {
	iq, ok := st.(*stanza.IQ)
	if !ok || iq.Type != stanza.IQGet {
		return nil
	}
	info := p.info
	res := stanza.IQ{Header: stanza.Header{ID: iq.ID, Type: stanza.IQResult, To: iq.From}}
	return p.params.SendElement(ctx, &stanza.IQPayload{IQ: res, Payload: &info})
}

func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

// Info returns the software version info.
func (p *Plugin) Info() Query {
	return p.info
}

func init() {
	_ = ns.Version
}
