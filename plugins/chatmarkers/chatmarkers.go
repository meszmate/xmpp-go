// Package chatmarkers implements XEP-0333 Chat Markers.
package chatmarkers

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "chatmarkers"

type Markable struct {
	XMLName xml.Name `xml:"urn:xmpp:chat-markers:0 markable"`
}
type Received struct {
	XMLName xml.Name `xml:"urn:xmpp:chat-markers:0 received"`
	ID      string   `xml:"id,attr"`
}
type Displayed struct {
	XMLName xml.Name `xml:"urn:xmpp:chat-markers:0 displayed"`
	ID      string   `xml:"id,attr"`
}
type Acknowledged struct {
	XMLName xml.Name `xml:"urn:xmpp:chat-markers:0 acknowledged"`
	ID      string   `xml:"id,attr"`
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

func init() { _ = ns.ChatMarkers }
