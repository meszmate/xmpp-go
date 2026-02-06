// Package version implements XEP-0092 Software Version.
package version

import (
	"context"
	"encoding/xml"
	"runtime"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "version"

// Query represents a software version query.
type Query struct {
	XMLName xml.Name `xml:"jabber:iq:version query"`
	Name    string   `xml:"name,omitempty"`
	Version string   `xml:"version,omitempty"`
	OS      string   `xml:"os,omitempty"`
}

// Plugin implements XEP-0092.
type Plugin struct {
	info   Query
	params plugin.InitParams
}

// New creates a new version plugin.
func New(name, version string) *Plugin {
	return &Plugin{
		info: Query{
			Name:    name,
			Version: version,
			OS:      runtime.GOOS,
		},
	}
}

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }

func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	return nil
}

func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

// Info returns the software version info.
func (p *Plugin) Info() Query {
	return p.info
}

func init() {
	_ = ns.Version
}
