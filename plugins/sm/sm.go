// Package sm implements XEP-0198 Stream Management.
package sm

import (
	"context"
	"encoding/xml"
	"sync"
	"sync/atomic"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "sm"

type Enable struct {
	XMLName xml.Name `xml:"urn:xmpp:sm:3 enable"`
	Resume  bool     `xml:"resume,attr,omitempty"`
}

type Enabled struct {
	XMLName  xml.Name `xml:"urn:xmpp:sm:3 enabled"`
	ID       string   `xml:"id,attr,omitempty"`
	Resume   bool     `xml:"resume,attr,omitempty"`
	Max      int      `xml:"max,attr,omitempty"`
	Location string   `xml:"location,attr,omitempty"`
}

type Resume struct {
	XMLName xml.Name `xml:"urn:xmpp:sm:3 resume"`
	H       uint32   `xml:"h,attr"`
	PrevID  string   `xml:"previd,attr"`
}

type Resumed struct {
	XMLName xml.Name `xml:"urn:xmpp:sm:3 resumed"`
	H       uint32   `xml:"h,attr"`
	PrevID  string   `xml:"previd,attr"`
}

type Ack struct {
	XMLName xml.Name `xml:"urn:xmpp:sm:3 a"`
	H       uint32   `xml:"h,attr"`
}

type Request struct {
	XMLName xml.Name `xml:"urn:xmpp:sm:3 r"`
}

type Plugin struct {
	mu       sync.Mutex
	inbound  atomic.Uint32
	outbound atomic.Uint32
	queue    [][]byte
	id       string
	params   plugin.InitParams
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

func (p *Plugin) InboundCount() uint32  { return p.inbound.Load() }
func (p *Plugin) OutboundCount() uint32 { return p.outbound.Load() }
func (p *Plugin) IncrementInbound()     { p.inbound.Add(1) }
func (p *Plugin) IncrementOutbound()    { p.outbound.Add(1) }

func (p *Plugin) Enqueue(data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queue = append(p.queue, data)
}

func (p *Plugin) Ack(h uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	diff := int(h) - int(p.outbound.Load()-uint32(len(p.queue)))
	if diff > 0 && diff <= len(p.queue) {
		p.queue = p.queue[diff:]
	}
}

func (p *Plugin) SetID(id string) { p.mu.Lock(); p.id = id; p.mu.Unlock() }
func (p *Plugin) ID() string      { p.mu.Lock(); defer p.mu.Unlock(); return p.id }

func init() { _ = ns.SM }
