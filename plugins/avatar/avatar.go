// Package avatar implements XEP-0084 User Avatar and XEP-0153 vCard-Based Avatars.
package avatar

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "avatar"

// Data represents avatar data (XEP-0084).
type Data struct {
	XMLName xml.Name `xml:"urn:xmpp:avatar:data data"`
	Value   string   `xml:",chardata"`
}

// Metadata represents avatar metadata (XEP-0084).
type Metadata struct {
	XMLName xml.Name      `xml:"urn:xmpp:avatar:metadata metadata"`
	Info    []MetadataInfo `xml:"info"`
}

type MetadataInfo struct {
	XMLName xml.Name `xml:"info"`
	Bytes   int      `xml:"bytes,attr"`
	Height  int      `xml:"height,attr,omitempty"`
	Width   int      `xml:"width,attr,omitempty"`
	ID      string   `xml:"id,attr"`
	Type    string   `xml:"type,attr"`
	URL     string   `xml:"url,attr,omitempty"`
}

// VCardUpdate represents XEP-0153 vCard-Based Avatars.
type VCardUpdate struct {
	XMLName xml.Name `xml:"vcard-temp:x:update x"`
	Photo   *string  `xml:"photo"`
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
	_ = ns.AvatarData
	_ = ns.AvatarMetadata
	_ = ns.VCardUpdate
}
