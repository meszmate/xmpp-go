// Package jingle implements XEP-0166 Jingle and related extensions.
package jingle

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "jingle"

// Actions
const (
	ActionSessionInitiate  = "session-initiate"
	ActionSessionAccept    = "session-accept"
	ActionSessionTerminate = "session-terminate"
	ActionContentAdd       = "content-add"
	ActionContentRemove    = "content-remove"
	ActionContentModify    = "content-modify"
	ActionTransportInfo    = "transport-info"
	ActionTransportReplace = "transport-replace"
	ActionTransportAccept  = "transport-accept"
	ActionTransportReject  = "transport-reject"
	ActionDescriptionInfo  = "description-info"
	ActionSessionInfo      = "session-info"
)

type Jingle struct {
	XMLName   xml.Name  `xml:"urn:xmpp:jingle:1 jingle"`
	Action    string    `xml:"action,attr"`
	Initiator string    `xml:"initiator,attr,omitempty"`
	Responder string    `xml:"responder,attr,omitempty"`
	SID       string    `xml:"sid,attr"`
	Contents  []Content `xml:"content"`
	Reason    *Reason   `xml:"reason,omitempty"`
}

type Content struct {
	XMLName     xml.Name `xml:"content"`
	Creator     string   `xml:"creator,attr"`
	Name        string   `xml:"name,attr"`
	Senders     string   `xml:"senders,attr,omitempty"`
	Disposition string   `xml:"disposition,attr,omitempty"`
	Description []byte   `xml:",innerxml"`
}

type Reason struct {
	XMLName   xml.Name `xml:"reason"`
	Condition string   `xml:"-"`
	Text      string   `xml:"text,omitempty"`
}

// RTP Description (XEP-0167)
type RTPDescription struct {
	XMLName      xml.Name      `xml:"urn:xmpp:jingle:apps:rtp:1 description"`
	Media        string        `xml:"media,attr"`
	PayloadTypes []PayloadType `xml:"payload-type"`
}

type PayloadType struct {
	XMLName    xml.Name    `xml:"payload-type"`
	ID         int         `xml:"id,attr"`
	Name       string      `xml:"name,attr"`
	Clockrate  int         `xml:"clockrate,attr,omitempty"`
	Channels   int         `xml:"channels,attr,omitempty"`
	Parameters []Parameter `xml:"parameter"`
}

type Parameter struct {
	XMLName xml.Name `xml:"parameter"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr"`
}

// ICE-UDP Transport (XEP-0176)
type ICEUDPTransport struct {
	XMLName     xml.Name     `xml:"urn:xmpp:jingle:transports:ice-udp:1 transport"`
	Ufrag       string       `xml:"ufrag,attr,omitempty"`
	Pwd         string       `xml:"pwd,attr,omitempty"`
	Candidates  []Candidate  `xml:"candidate"`
	Fingerprint *Fingerprint `xml:"urn:xmpp:jingle:apps:dtls:0 fingerprint,omitempty"`
}

type Candidate struct {
	XMLName    xml.Name `xml:"candidate"`
	Component  int      `xml:"component,attr"`
	Foundation string   `xml:"foundation,attr"`
	Generation int      `xml:"generation,attr"`
	ID         string   `xml:"id,attr"`
	IP         string   `xml:"ip,attr"`
	Network    int      `xml:"network,attr,omitempty"`
	Port       int      `xml:"port,attr"`
	Priority   int      `xml:"priority,attr"`
	Protocol   string   `xml:"protocol,attr"`
	Type       string   `xml:"type,attr"`
	RelAddr    string   `xml:"rel-addr,attr,omitempty"`
	RelPort    int      `xml:"rel-port,attr,omitempty"`
}

// DTLS-SRTP (XEP-0320)
type Fingerprint struct {
	XMLName xml.Name `xml:"urn:xmpp:jingle:apps:dtls:0 fingerprint"`
	Hash    string   `xml:"hash,attr"`
	Setup   string   `xml:"setup,attr"`
	Value   string   `xml:",chardata"`
}

// Raw UDP Transport (XEP-0177)
type RawUDPTransport struct {
	XMLName    xml.Name       `xml:"urn:xmpp:jingle:transports:raw-udp:1 transport"`
	Candidates []RawCandidate `xml:"candidate"`
}

type RawCandidate struct {
	XMLName    xml.Name `xml:"candidate"`
	Component  int      `xml:"component,attr"`
	Generation int      `xml:"generation,attr"`
	ID         string   `xml:"id,attr"`
	IP         string   `xml:"ip,attr"`
	Port       int      `xml:"port,attr"`
	Type       string   `xml:"type,attr,omitempty"`
}

// Jingle Message Initiation (XEP-0353)
type Propose struct {
	XMLName      xml.Name      `xml:"urn:xmpp:jingle-message:0 propose"`
	ID           string        `xml:"id,attr"`
	Descriptions []ProposeDesc `xml:"description"`
}

type ProposeDesc struct {
	XMLName xml.Name `xml:"description"`
	Media   string   `xml:"media,attr"`
	NS      string   `xml:"xmlns,attr"`
}

type Retract struct {
	XMLName xml.Name `xml:"urn:xmpp:jingle-message:0 retract"`
	ID      string   `xml:"id,attr"`
}

type Accept struct {
	XMLName xml.Name `xml:"urn:xmpp:jingle-message:0 accept"`
	ID      string   `xml:"id,attr"`
}

type Reject struct {
	XMLName xml.Name `xml:"urn:xmpp:jingle-message:0 reject"`
	ID      string   `xml:"id,attr"`
}

type Proceed struct {
	XMLName xml.Name `xml:"urn:xmpp:jingle-message:0 proceed"`
	ID      string   `xml:"id,attr"`
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
	_ = ns.Jingle
	_ = ns.JingleRTP
	_ = ns.JingleICEUDP
	_ = ns.JingleRawUDP
	_ = ns.JingleDTLS
	_ = ns.JingleMI
}
