// Package vcard implements XEP-0054 vcard-temp and XEP-0292 vCard4 over XMPP.
package vcard

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "vcard"

// VCard represents a vcard-temp (XEP-0054).
type VCard struct {
	XMLName  xml.Name `xml:"vcard-temp vCard"`
	FN       string   `xml:"FN,omitempty"`
	N        *Name_   `xml:"N,omitempty"`
	Nickname string   `xml:"NICKNAME,omitempty"`
	Email    *Email   `xml:"EMAIL,omitempty"`
	URL      string   `xml:"URL,omitempty"`
	Photo    *Photo   `xml:"PHOTO,omitempty"`
	Bday     string   `xml:"BDAY,omitempty"`
	Org      *Org     `xml:"ORG,omitempty"`
	Title    string   `xml:"TITLE,omitempty"`
	Desc     string   `xml:"DESC,omitempty"`
}

type Name_ struct {
	Family string `xml:"FAMILY,omitempty"`
	Given  string `xml:"GIVEN,omitempty"`
	Middle string `xml:"MIDDLE,omitempty"`
}

type Email struct {
	UserID string `xml:"USERID,omitempty"`
}

type Photo struct {
	Type   string `xml:"TYPE,omitempty"`
	BinVal string `xml:"BINVAL,omitempty"`
	ExtVal string `xml:"EXTVAL,omitempty"`
}

type Org struct {
	OrgName string `xml:"ORGNAME,omitempty"`
	OrgUnit string `xml:"ORGUNIT,omitempty"`
}

// VCard4 represents a vCard4 (XEP-0292).
type VCard4 struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:vcard-4.0 vcard"`
	FN      *Text    `xml:"fn,omitempty"`
	N       *VCard4N `xml:"n,omitempty"`
	Email   *Text    `xml:"email,omitempty"`
	URL     *URI     `xml:"url,omitempty"`
	Nickname *Text   `xml:"nickname,omitempty"`
}

type VCard4N struct {
	Surname string `xml:"surname,omitempty"`
	Given   string `xml:"given,omitempty"`
}

type Text struct {
	Text string `xml:"text"`
}

type URI struct {
	URI string `xml:"uri"`
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
	_ = ns.VCard
	_ = ns.VCard4
}
