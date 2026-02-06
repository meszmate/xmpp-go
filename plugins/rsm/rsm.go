// Package rsm implements XEP-0059 Result Set Management.
package rsm

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "rsm"

// Set represents a result set management element.
type Set struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/rsm set"`
	After   string   `xml:"after,omitempty"`
	Before  string   `xml:"before,omitempty"`
	Count   *int     `xml:"count,omitempty"`
	First   *First   `xml:"first,omitempty"`
	Index   *int     `xml:"index,omitempty"`
	Last    string   `xml:"last,omitempty"`
	Max     *int     `xml:"max,omitempty"`
}

// First represents the first element with an index attribute.
type First struct {
	XMLName xml.Name `xml:"first"`
	Index   int      `xml:"index,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

// Plugin implements XEP-0059.
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

// NewRequest creates a new RSM request with the given max.
func NewRequest(max int) Set {
	return Set{Max: &max}
}

// NewRequestAfter creates a request for items after the given ID.
func NewRequestAfter(max int, after string) Set {
	return Set{Max: &max, After: after}
}

// NewRequestBefore creates a request for items before the given ID.
func NewRequestBefore(max int, before string) Set {
	return Set{Max: &max, Before: before}
}

func init() { _ = ns.RSM }
