// Package csi implements XEP-0352 Client State Indication.
package csi

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "csi"

type Active struct {
	XMLName xml.Name `xml:"urn:xmpp:csi:0 active"`
}

type Inactive struct {
	XMLName xml.Name `xml:"urn:xmpp:csi:0 inactive"`
}

type Plugin struct {
	active bool
	params plugin.InitParams
}

func New() *Plugin { return &Plugin{active: true} }

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }
func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	return nil
}
func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

func (p *Plugin) IsActive() bool   { return p.active }
func (p *Plugin) SetActive(v bool) { p.active = v }

func init() { _ = ns.CSI }
