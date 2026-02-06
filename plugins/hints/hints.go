// Package hints implements XEP-0334 Message Processing Hints.
package hints

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "hints"

type NoPermanentStore struct {
	XMLName xml.Name `xml:"urn:xmpp:hints no-permanent-store"`
}
type NoStore struct {
	XMLName xml.Name `xml:"urn:xmpp:hints no-store"`
}
type NoCopy struct {
	XMLName xml.Name `xml:"urn:xmpp:hints no-copy"`
}
type Store struct {
	XMLName xml.Name `xml:"urn:xmpp:hints store"`
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

func init() { _ = ns.Hints }
