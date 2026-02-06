// Package extdisco implements XEP-0215 External Service Discovery.
package extdisco

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "extdisco"

type Services struct {
	XMLName  xml.Name  `xml:"urn:xmpp:extdisco:2 services"`
	Type     string    `xml:"type,attr,omitempty"`
	Services []Service `xml:"service"`
}

type Service struct {
	XMLName    xml.Name `xml:"service"`
	Host       string   `xml:"host,attr"`
	Port       int      `xml:"port,attr,omitempty"`
	Transport  string   `xml:"transport,attr,omitempty"`
	Type       string   `xml:"type,attr"`
	Name       string   `xml:"name,attr,omitempty"`
	Username   string   `xml:"username,attr,omitempty"`
	Password   string   `xml:"password,attr,omitempty"`
	Restricted bool     `xml:"restricted,attr,omitempty"`
	Expires    string   `xml:"expires,attr,omitempty"`
}

type Credentials struct {
	XMLName xml.Name `xml:"urn:xmpp:extdisco:2 credentials"`
	Service *Service `xml:"service"`
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

func init() { _ = ns.ExtDisco }
