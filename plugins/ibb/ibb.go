// Package ibb implements XEP-0047 In-Band Bytestreams.
package ibb

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "ibb"

type Open struct {
	XMLName   xml.Name `xml:"http://jabber.org/protocol/ibb open"`
	BlockSize int      `xml:"block-size,attr"`
	SID       string   `xml:"sid,attr"`
	Stanza    string   `xml:"stanza,attr,omitempty"`
}

type Data struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/ibb data"`
	SID     string   `xml:"sid,attr"`
	Seq     uint16   `xml:"seq,attr"`
	Value   string   `xml:",chardata"`
}

type Close struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/ibb close"`
	SID     string   `xml:"sid,attr"`
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

func init() { _ = ns.IBB }
