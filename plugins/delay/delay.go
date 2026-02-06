// Package delay implements XEP-0203 Delayed Delivery.
package delay

import (
	"context"
	"encoding/xml"
	gotime "time"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "delay"

// Delay represents a delayed delivery element.
type Delay struct {
	XMLName xml.Name `xml:"urn:xmpp:delay delay"`
	From    string   `xml:"from,attr,omitempty"`
	Stamp   string   `xml:"stamp,attr"`
	Reason  string   `xml:",chardata"`
}

// Plugin implements XEP-0203.
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

// NewDelay creates a Delay element with the given from and timestamp.
func NewDelay(from string, stamp gotime.Time) Delay {
	return Delay{
		From:  from,
		Stamp: stamp.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// ParseStamp parses the stamp attribute.
func (d Delay) ParseStamp() (gotime.Time, error) {
	return gotime.Parse("2006-01-02T15:04:05Z", d.Stamp)
}

func init() { _ = ns.Delay }
