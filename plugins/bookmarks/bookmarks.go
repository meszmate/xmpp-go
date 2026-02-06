// Package bookmarks implements XEP-0402 PEP Native Bookmarks.
package bookmarks

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "bookmarks"

// PEP node for bookmarks.
const Node = "urn:xmpp:bookmarks:1"

type Conference struct {
	XMLName    xml.Name    `xml:"urn:xmpp:bookmarks:1 conference"`
	Autojoin   bool        `xml:"autojoin,attr,omitempty"`
	Name       string      `xml:"name,attr,omitempty"`
	Nick       string      `xml:"nick,omitempty"`
	Password   string      `xml:"password,omitempty"`
	Extensions []Extension `xml:"extensions,omitempty"`
}

type Extension struct {
	XMLName xml.Name
	Inner   []byte `xml:",innerxml"`
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

func init() { _ = ns.Bookmarks }
