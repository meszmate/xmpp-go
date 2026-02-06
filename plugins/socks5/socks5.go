// Package socks5 implements XEP-0065 SOCKS5 Bytestreams.
package socks5

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "socks5"

type Query struct {
	XMLName     xml.Name        `xml:"http://jabber.org/protocol/bytestreams query"`
	SID         string          `xml:"sid,attr"`
	Mode        string          `xml:"mode,attr,omitempty"`
	Streamhosts []Streamhost    `xml:"streamhost"`
	Used        *StreamhostUsed `xml:"streamhost-used,omitempty"`
}

type Streamhost struct {
	XMLName xml.Name `xml:"streamhost"`
	JID     string   `xml:"jid,attr"`
	Host    string   `xml:"host,attr"`
	Port    int      `xml:"port,attr"`
}

type StreamhostUsed struct {
	XMLName xml.Name `xml:"streamhost-used"`
	JID     string   `xml:"jid,attr"`
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

func init() { _ = ns.SOCKS5 }
