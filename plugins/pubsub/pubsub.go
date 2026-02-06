// Package pubsub implements XEP-0060 Publish-Subscribe and XEP-0163 PEP.
package pubsub

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "pubsub"

type PubSub struct {
	XMLName     xml.Name     `xml:"http://jabber.org/protocol/pubsub pubsub"`
	Create      *Create      `xml:"create,omitempty"`
	Configure   *Configure   `xml:"configure,omitempty"`
	Subscribe   *SubReq      `xml:"subscribe,omitempty"`
	Unsubscribe *Unsub       `xml:"unsubscribe,omitempty"`
	Publish     *Publish     `xml:"publish,omitempty"`
	Retract     *Retract     `xml:"retract,omitempty"`
	Items       *Items       `xml:"items,omitempty"`
	Subscription *Subscription `xml:"subscription,omitempty"`
}

type Create struct {
	XMLName xml.Name `xml:"create"`
	Node    string   `xml:"node,attr,omitempty"`
}

type Configure struct {
	XMLName xml.Name `xml:"configure"`
	Form    []byte   `xml:",innerxml"`
}

type SubReq struct {
	XMLName xml.Name `xml:"subscribe"`
	Node    string   `xml:"node,attr"`
	JID     string   `xml:"jid,attr"`
}

type Unsub struct {
	XMLName xml.Name `xml:"unsubscribe"`
	Node    string   `xml:"node,attr"`
	JID     string   `xml:"jid,attr"`
	SubID   string   `xml:"subid,attr,omitempty"`
}

type Publish struct {
	XMLName xml.Name `xml:"publish"`
	Node    string   `xml:"node,attr"`
	Items   []PubItem `xml:"item"`
}

type PubItem struct {
	XMLName xml.Name `xml:"item"`
	ID      string   `xml:"id,attr,omitempty"`
	Payload []byte   `xml:",innerxml"`
}

type Retract struct {
	XMLName xml.Name `xml:"retract"`
	Node    string   `xml:"node,attr"`
	Notify  bool     `xml:"notify,attr,omitempty"`
	Items   []PubItem `xml:"item"`
}

type Items struct {
	XMLName xml.Name  `xml:"items"`
	Node    string    `xml:"node,attr"`
	SubID   string    `xml:"subid,attr,omitempty"`
	MaxItems *int     `xml:"max_items,attr,omitempty"`
	Items   []PubItem `xml:"item"`
}

type Subscription struct {
	XMLName xml.Name `xml:"subscription"`
	Node    string   `xml:"node,attr"`
	JID     string   `xml:"jid,attr"`
	SubID   string   `xml:"subid,attr,omitempty"`
	State   string   `xml:"subscription,attr,omitempty"`
}

// Event is from PubSub event notifications.
type Event struct {
	XMLName xml.Name     `xml:"http://jabber.org/protocol/pubsub#event event"`
	Items   *EventItems  `xml:"items,omitempty"`
	Purge   *EventPurge  `xml:"purge,omitempty"`
	Delete  *EventDelete `xml:"delete,omitempty"`
}

type EventItems struct {
	XMLName xml.Name  `xml:"items"`
	Node    string    `xml:"node,attr"`
	Items   []PubItem `xml:"item"`
	Retract []EventRetract `xml:"retract"`
}

type EventRetract struct {
	XMLName xml.Name `xml:"retract"`
	ID      string   `xml:"id,attr"`
}

type EventPurge struct {
	XMLName xml.Name `xml:"purge"`
	Node    string   `xml:"node,attr"`
}

type EventDelete struct {
	XMLName xml.Name `xml:"delete"`
	Node    string   `xml:"node,attr"`
}

// Owner types
type PubSubOwner struct {
	XMLName   xml.Name   `xml:"http://jabber.org/protocol/pubsub#owner pubsub"`
	Configure *OwnerConfigure `xml:"configure,omitempty"`
	Delete    *OwnerDelete    `xml:"delete,omitempty"`
	Purge     *OwnerPurge     `xml:"purge,omitempty"`
}

type OwnerConfigure struct {
	XMLName xml.Name `xml:"configure"`
	Node    string   `xml:"node,attr"`
	Form    []byte   `xml:",innerxml"`
}

type OwnerDelete struct {
	XMLName xml.Name `xml:"delete"`
	Node    string   `xml:"node,attr"`
}

type OwnerPurge struct {
	XMLName xml.Name `xml:"purge"`
	Node    string   `xml:"node,attr"`
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

func init() {
	_ = ns.PubSub
	_ = ns.PubSubEvent
	_ = ns.PubSubOwner
}
