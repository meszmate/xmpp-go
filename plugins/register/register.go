// Package register implements XEP-0077 In-Band Registration.
package register

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "register"

type Query struct {
	XMLName      xml.Name `xml:"jabber:iq:register query"`
	Registered   *Empty   `xml:"registered,omitempty"`
	Username     string   `xml:"username,omitempty"`
	Password     string   `xml:"password,omitempty"`
	Email        string   `xml:"email,omitempty"`
	Instructions string   `xml:"instructions,omitempty"`
	Remove       *Empty   `xml:"remove,omitempty"`
	Form         []byte   `xml:",innerxml"`
}

type Empty struct {
	XMLName xml.Name
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

func init() { _ = ns.Register }
