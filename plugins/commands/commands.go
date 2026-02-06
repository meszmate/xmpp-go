// Package commands implements XEP-0050 Ad-Hoc Commands.
package commands

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "commands"

const (
	StatusExecuting = "executing"
	StatusCompleted = "completed"
	StatusCanceled  = "canceled"
)

const (
	ActionExecute  = "execute"
	ActionCancel   = "cancel"
	ActionPrev     = "prev"
	ActionNext     = "next"
	ActionComplete = "complete"
)

type Command struct {
	XMLName   xml.Name `xml:"http://jabber.org/protocol/commands command"`
	Node      string   `xml:"node,attr"`
	SessionID string   `xml:"sessionid,attr,omitempty"`
	Action    string   `xml:"action,attr,omitempty"`
	Status    string   `xml:"status,attr,omitempty"`
	Actions   *Actions `xml:"actions,omitempty"`
	Note      *Note    `xml:"note,omitempty"`
	Form      []byte   `xml:",innerxml"`
}

type Actions struct {
	XMLName  xml.Name `xml:"actions"`
	Execute  string   `xml:"execute,attr,omitempty"`
	Prev     *Empty   `xml:"prev,omitempty"`
	Next     *Empty   `xml:"next,omitempty"`
	Complete *Empty   `xml:"complete,omitempty"`
}

type Note struct {
	XMLName xml.Name `xml:"note"`
	Type    string   `xml:"type,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

type Empty struct{}

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

func init() { _ = ns.Commands }
