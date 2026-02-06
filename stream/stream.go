// Package stream provides XMPP stream header types and stream management.
package stream

import (
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/jid"
)

// Header represents an XMPP stream header.
type Header struct {
	XMLName xml.Name `xml:"http://etherx.jabber.org/streams stream"`
	To      jid.JID  `xml:"to,attr,omitempty"`
	From    jid.JID  `xml:"from,attr,omitempty"`
	ID      string   `xml:"id,attr,omitempty"`
	Version string   `xml:"version,attr,omitempty"`
	Lang    string   `xml:"xml:lang,attr,omitempty"`
	NS      string   `xml:"xmlns,attr,omitempty"`
}

// DefaultVersion is the default XMPP stream version.
const DefaultVersion = "1.0"

// Open returns the XML bytes for opening a stream.
func Open(h Header) []byte {
	if h.Version == "" {
		h.Version = DefaultVersion
	}
	if h.NS == "" {
		h.NS = ns.Client
	}

	var buf []byte
	buf = append(buf, `<?xml version='1.0'?><stream:stream`...)

	if h.To.String() != "" {
		buf = append(buf, ` to='`...)
		buf = append(buf, h.To.String()...)
		buf = append(buf, '\'')
	}
	if h.From.String() != "" {
		buf = append(buf, ` from='`...)
		buf = append(buf, h.From.String()...)
		buf = append(buf, '\'')
	}
	if h.ID != "" {
		buf = append(buf, ` id='`...)
		buf = append(buf, h.ID...)
		buf = append(buf, '\'')
	}

	buf = append(buf, ` version='`...)
	buf = append(buf, h.Version...)
	buf = append(buf, '\'')

	if h.Lang != "" {
		buf = append(buf, ` xml:lang='`...)
		buf = append(buf, h.Lang...)
		buf = append(buf, '\'')
	}

	buf = append(buf, ` xmlns='`...)
	buf = append(buf, h.NS...)
	buf = append(buf, '\'')

	buf = append(buf, ` xmlns:stream='`...)
	buf = append(buf, ns.Stream...)
	buf = append(buf, `'>`...)

	return buf
}

// Close returns the XML bytes for closing a stream.
func Close() []byte {
	return []byte(`</stream:stream>`)
}

// Features represents the stream features element.
type Features struct {
	XMLName xml.Name `xml:"http://etherx.jabber.org/streams features"`
}

// WebSocketOpen represents a WebSocket XMPP open frame (RFC 7395).
type WebSocketOpen struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-framing open"`
	To      string   `xml:"to,attr,omitempty"`
	From    string   `xml:"from,attr,omitempty"`
	ID      string   `xml:"id,attr,omitempty"`
	Version string   `xml:"version,attr,omitempty"`
	Lang    string   `xml:"xml:lang,attr,omitempty"`
}

// WebSocketClose represents a WebSocket XMPP close frame (RFC 7395).
type WebSocketClose struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-framing close"`
}
