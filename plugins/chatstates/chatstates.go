// Package chatstates implements XEP-0085 Chat State Notifications.
package chatstates

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "chatstates"

const (
	StateActive    = "active"
	StateComposing = "composing"
	StatePaused    = "paused"
	StateInactive  = "inactive"
	StateGone      = "gone"
)

type Active struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/chatstates active"`
}
type Composing struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/chatstates composing"`
}
type Paused struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/chatstates paused"`
}
type Inactive struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/chatstates inactive"`
}
type Gone struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/chatstates gone"`
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

func init() { _ = ns.ChatStates }
