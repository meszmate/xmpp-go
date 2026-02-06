// Package mix implements XEP-0369 MIX and related extensions (XEP-0403/0405/0406/0407).
package mix

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "mix"

// Well-known MIX nodes.
const (
	NodeMessages     = "urn:xmpp:mix:nodes:messages"
	NodePresence     = "urn:xmpp:mix:nodes:presence"
	NodeParticipants = "urn:xmpp:mix:nodes:participants"
	NodeInfo         = "urn:xmpp:mix:nodes:info"
	NodeConfig       = "urn:xmpp:mix:nodes:config"
	NodeAllowed      = "urn:xmpp:mix:nodes:allowed"
	NodeBanned       = "urn:xmpp:mix:nodes:banned"
)

type Join struct {
	XMLName   xml.Name `xml:"urn:xmpp:mix:core:1 join"`
	ID        string   `xml:"id,attr,omitempty"`
	Nick      string   `xml:"nick,omitempty"`
	Subscribe []Subscribe `xml:"subscribe"`
}

type Leave struct {
	XMLName xml.Name `xml:"urn:xmpp:mix:core:1 leave"`
}

type Subscribe struct {
	XMLName xml.Name `xml:"subscribe"`
	Node    string   `xml:"node,attr"`
}

type Participant struct {
	XMLName xml.Name `xml:"urn:xmpp:mix:core:1 participant"`
	Nick    string   `xml:"nick,omitempty"`
	JID     string   `xml:"jid,omitempty"`
}

type SetNick struct {
	XMLName xml.Name `xml:"urn:xmpp:mix:core:1 setnick"`
	Nick    string   `xml:"nick"`
}

type Create struct {
	XMLName xml.Name `xml:"urn:xmpp:mix:core:1 create"`
	Channel string   `xml:"channel,attr,omitempty"`
}

type Destroy struct {
	XMLName xml.Name `xml:"urn:xmpp:mix:core:1 destroy"`
	Channel string   `xml:"channel,attr"`
}

// ClientJoin is from XEP-0405 (MIX-PAM).
type ClientJoin struct {
	XMLName xml.Name `xml:"urn:xmpp:mix:pam:2 client-join"`
	Channel string   `xml:"channel,attr"`
	Join    *Join    `xml:"urn:xmpp:mix:core:1 join"`
}

type ClientLeave struct {
	XMLName xml.Name `xml:"urn:xmpp:mix:pam:2 client-leave"`
	Channel string   `xml:"channel,attr"`
	Leave   *Leave   `xml:"urn:xmpp:mix:core:1 leave"`
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
	_ = ns.MIXCore
	_ = ns.MIXPAM
}
