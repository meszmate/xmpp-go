// Package sasl2 implements XEP-0388 SASL2, XEP-0484 FAST, XEP-0386 Bind2, and XEP-0440 SASL Channel-Binding.
package sasl2

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "sasl2"

// SASL2 (XEP-0388)
type Authentication struct {
	XMLName    xml.Name    `xml:"urn:xmpp:sasl:2 authentication"`
	Mechanisms []Mechanism `xml:"mechanism"`
	Inline     *Inline     `xml:"inline,omitempty"`
}

type Mechanism struct {
	XMLName xml.Name `xml:"mechanism"`
	Value   string   `xml:",chardata"`
}

type Authenticate struct {
	XMLName         xml.Name   `xml:"urn:xmpp:sasl:2 authenticate"`
	Mechanism       string     `xml:"mechanism,attr"`
	InitialResponse string     `xml:"initial-response,omitempty"`
	UserAgent       *UserAgent `xml:"user-agent,omitempty"`
	Inline          []byte     `xml:",innerxml"`
}

type UserAgent struct {
	XMLName  xml.Name `xml:"user-agent"`
	ID       string   `xml:"id,attr,omitempty"`
	Software string   `xml:"software,omitempty"`
	Device   string   `xml:"device,omitempty"`
}

type Challenge struct {
	XMLName xml.Name `xml:"urn:xmpp:sasl:2 challenge"`
	Value   string   `xml:",chardata"`
}

type Response struct {
	XMLName xml.Name `xml:"urn:xmpp:sasl:2 response"`
	Value   string   `xml:",chardata"`
}

type Success struct {
	XMLName        xml.Name `xml:"urn:xmpp:sasl:2 success"`
	AdditionalData string   `xml:"additional-data,omitempty"`
	AuthzID        string   `xml:"authorization-identifier,omitempty"`
	Inner          []byte   `xml:",innerxml"`
}

type Failure struct {
	XMLName   xml.Name `xml:"urn:xmpp:sasl:2 failure"`
	Condition string   `xml:"-"`
	Text      string   `xml:"text,omitempty"`
}

type Inline struct {
	XMLName xml.Name `xml:"inline"`
	Inner   []byte   `xml:",innerxml"`
}

// Bind2 (XEP-0386)
type Bind2 struct {
	XMLName xml.Name `xml:"urn:xmpp:bind:0 bind"`
	Tag     string   `xml:"tag,omitempty"`
	Inner   []byte   `xml:",innerxml"`
}

type Bound struct {
	XMLName xml.Name `xml:"urn:xmpp:bind:0 bound"`
	Inner   []byte   `xml:",innerxml"`
}

// FAST (XEP-0484)
type Fast struct {
	XMLName    xml.Name    `xml:"urn:xmpp:fast:0 fast"`
	Mechanisms []Mechanism `xml:"mechanism"`
	Invalidate bool        `xml:"invalidate,attr,omitempty"`
}

type RequestToken struct {
	XMLName   xml.Name `xml:"urn:xmpp:fast:0 request-token"`
	Mechanism string   `xml:"mechanism,attr"`
}

type Token struct {
	XMLName xml.Name `xml:"urn:xmpp:fast:0 token"`
	Token   string   `xml:"token,attr"`
	Expiry  string   `xml:"expiry,attr,omitempty"`
}

// Channel-Binding (XEP-0440)
type SASLChannelBinding struct {
	XMLName         xml.Name         `xml:"urn:xmpp:sasl-cb:0 sasl-channel-binding"`
	ChannelBindings []ChannelBinding `xml:"channel-binding"`
}

type ChannelBinding struct {
	XMLName xml.Name `xml:"channel-binding"`
	Type    string   `xml:"type,attr"`
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
	_ = ns.SASL2
	_ = ns.FAST
	_ = ns.Bind2
	_ = ns.SASLCBind
}
