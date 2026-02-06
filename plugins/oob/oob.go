// Package oob implements XEP-0066 Out of Band Data.
package oob

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "oob"

// X represents an OOB element in a message (jabber:x:oob).
type X struct {
	XMLName xml.Name `xml:"jabber:x:oob x"`
	URL     string   `xml:"url"`
	Desc    string   `xml:"desc,omitempty"`
}

// Query represents an OOB IQ query (jabber:iq:oob).
type Query struct {
	XMLName xml.Name `xml:"jabber:iq:oob query"`
	URL     string   `xml:"url"`
	Desc    string   `xml:"desc,omitempty"`
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
	_ = ns.OOB
	_ = ns.OOB2
}
