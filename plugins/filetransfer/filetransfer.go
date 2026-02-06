// Package filetransfer implements XEP-0234 Jingle File Transfer and XEP-0446/0447/0448 Stateless File Sharing.
package filetransfer

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "filetransfer"

// Jingle File Transfer (XEP-0234)
type Description struct {
	XMLName xml.Name `xml:"urn:xmpp:jingle:apps:file-transfer:5 description"`
	File    *File    `xml:"file"`
}

type File struct {
	XMLName   xml.Name `xml:"file"`
	Date      string   `xml:"date,omitempty"`
	Desc      string   `xml:"desc,omitempty"`
	MediaType string   `xml:"media-type,omitempty"`
	Name      string   `xml:"name,omitempty"`
	Size      int64    `xml:"size,omitempty"`
	Range     *Range   `xml:"range,omitempty"`
	Hashes    []Hash   `xml:"urn:xmpp:hashes:2 hash,omitempty"`
}

type Range struct {
	XMLName xml.Name `xml:"range"`
	Offset  int64    `xml:"offset,attr,omitempty"`
	Length  int64    `xml:"length,attr,omitempty"`
}

type Hash struct {
	XMLName xml.Name `xml:"urn:xmpp:hashes:2 hash"`
	Algo    string   `xml:"algo,attr"`
	Value   string   `xml:",chardata"`
}

// Stateless File Sharing (XEP-0447)
type FileSharing struct {
	XMLName      xml.Name       `xml:"urn:xmpp:sfs:0 file-sharing"`
	Disposition  string         `xml:"disposition,attr,omitempty"`
	FileMetadata *FileMetadataEl `xml:"urn:xmpp:file:metadata:0 file"`
	Sources      []Source       `xml:"sources>url-data"`
}

type FileMetadataEl struct {
	XMLName   xml.Name `xml:"urn:xmpp:file:metadata:0 file"`
	MediaType string   `xml:"media-type,omitempty"`
	Name      string   `xml:"name,omitempty"`
	Size      int64    `xml:"size,omitempty"`
	Desc      string   `xml:"desc,omitempty"`
	Hashes    []Hash   `xml:"urn:xmpp:hashes:2 hash,omitempty"`
}

type Source struct {
	XMLName xml.Name `xml:"url-data"`
	Target  string   `xml:"target,attr"`
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
	_ = ns.JingleFT
	_ = ns.FileMetadata
	_ = ns.SFS
}
