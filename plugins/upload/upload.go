// Package upload implements XEP-0363 HTTP File Upload.
package upload

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "upload"

type Request struct {
	XMLName     xml.Name `xml:"urn:xmpp:http:upload:0 request"`
	Filename    string   `xml:"filename,attr"`
	Size        int64    `xml:"size,attr"`
	ContentType string   `xml:"content-type,attr,omitempty"`
}

type Slot struct {
	XMLName xml.Name `xml:"urn:xmpp:http:upload:0 slot"`
	Put     Put      `xml:"put"`
	Get     Get      `xml:"get"`
}

type Put struct {
	XMLName xml.Name  `xml:"put"`
	URL     string    `xml:"url,attr"`
	Headers []Header  `xml:"header"`
}

type Get struct {
	XMLName xml.Name `xml:"get"`
	URL     string   `xml:"url,attr"`
}

type Header struct {
	XMLName xml.Name `xml:"header"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:",chardata"`
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

func init() { _ = ns.HTTPUpload }
