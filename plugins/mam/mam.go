// Package mam implements XEP-0313 Message Archive Management.
package mam

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "mam"

type Query struct {
	XMLName xml.Name `xml:"urn:xmpp:mam:2 query"`
	QueryID string   `xml:"queryid,attr,omitempty"`
	Node    string   `xml:"node,attr,omitempty"`
	Form    []byte   `xml:",innerxml"`
}

type Fin struct {
	XMLName  xml.Name `xml:"urn:xmpp:mam:2 fin"`
	Complete bool     `xml:"complete,attr,omitempty"`
	Stable   bool     `xml:"stable,attr,omitempty"`
	Set      []byte   `xml:",innerxml"`
}

type Result struct {
	XMLName   xml.Name `xml:"urn:xmpp:mam:2 result"`
	QueryID   string   `xml:"queryid,attr,omitempty"`
	ID        string   `xml:"id,attr"`
	Forwarded []byte   `xml:",innerxml"`
}

type Prefs struct {
	XMLName xml.Name `xml:"urn:xmpp:mam:2 prefs"`
	Default string   `xml:"default,attr"`
	Always  *JIDList `xml:"always,omitempty"`
	Never   *JIDList `xml:"never,omitempty"`
}

type JIDList struct {
	JIDs []string `xml:"jid"`
}

type Metadata struct {
	XMLName xml.Name `xml:"urn:xmpp:mam:2 metadata"`
	Start   *Info    `xml:"start,omitempty"`
	End     *Info    `xml:"end,omitempty"`
}

type Info struct {
	ID        string `xml:"id,attr"`
	Timestamp string `xml:"timestamp,attr"`
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

func init() { _ = ns.MAM }
