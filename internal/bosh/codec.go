// Package bosh implements the framing codec shared by the BOSH (XEP-0124/0206)
// client transport and server connection manager: it splits a continuous XMPP
// byte stream into discrete top-level elements and wraps/unwraps them in HTTP
// <body/> envelopes.
package bosh

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Protocol namespaces and prefixes (XEP-0124 / XEP-0206).
const (
	NS         = "http://jabber.org/protocol/httpbind"
	XMPPNS     = "urn:xmpp:xbosh"
	XMPPPrefix = "xmpp"
	StreamNS   = "http://etherx.jabber.org/streams"
	ClientNS   = "jabber:client"
)

// EventKind classifies a parsed stream event.
type EventKind int

const (
	// Open is a <stream:stream> header.
	Open EventKind = iota
	// Element is a complete top-level stanza or nonza.
	Element
	// Close is </stream:stream>.
	Close
)

// Event is one item pulled from an XMPP byte stream.
type Event struct {
	Kind  EventKind
	Attrs map[string]string
	Raw   []byte
}

// recordingReader tees everything read into an offset-indexed buffer so the
// parser can slice out an element's exact bytes via xml.Decoder input offsets.
type recordingReader struct {
	r    io.Reader
	buf  []byte
	base int64
}

func (rr *recordingReader) Read(p []byte) (int, error) {
	n, err := rr.r.Read(p)
	if n > 0 {
		rr.buf = append(rr.buf, p[:n]...)
	}
	return n, err
}

func (rr *recordingReader) slice(start, end int64) []byte {
	lo := start - rr.base
	hi := end - rr.base
	if lo < 0 {
		lo = 0
	}
	if hi > int64(len(rr.buf)) {
		hi = int64(len(rr.buf))
	}
	if lo > hi {
		lo = hi
	}
	out := make([]byte, hi-lo)
	copy(out, rr.buf[lo:hi])
	return out
}

func (rr *recordingReader) discard(upTo int64) {
	drop := upTo - rr.base
	if drop <= 0 {
		return
	}
	if drop > int64(len(rr.buf)) {
		drop = int64(len(rr.buf))
	}
	rr.buf = append([]byte(nil), rr.buf[drop:]...)
	rr.base += drop
}

// Parser reads an XMPP stream — which may contain multiple <stream:stream>
// re-opens produced by stream restarts — and yields Events. It uses RawToken so
// consecutive root elements do not trip the cooked decoder's "second root"
// check.
type Parser struct {
	rec        *recordingReader
	dec        *xml.Decoder
	depth      int
	elemStart  int64
	prevOffset int64
}

// NewParser creates a Parser over r.
func NewParser(r io.Reader) *Parser {
	rec := &recordingReader{r: r}
	return &Parser{rec: rec, dec: xml.NewDecoder(rec)}
}

// Next returns the next stream event, blocking until enough bytes are available,
// and io.EOF when the stream ends.
func (p *Parser) Next() (Event, error) {
	for {
		p.prevOffset = p.dec.InputOffset()
		tok, err := p.dec.RawToken()
		if err != nil {
			return Event{}, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			// A <stream:stream> is a stream (re)open at any depth: XMPP restarts
			// (after STARTTLS/SASL) send a fresh header without closing the prior
			// stream, so it can appear while we still believe we are inside the
			// previous stream. Treat it as a reset to stream scope.
			if isStreamOpen(t) {
				attrs := attrMap(t)
				p.depth = 1
				p.rec.discard(p.dec.InputOffset())
				return Event{Kind: Open, Attrs: attrs}, nil
			}
			if p.depth == 1 {
				p.elemStart = p.prevOffset
			}
			p.depth++
		case xml.EndElement:
			if p.depth == 1 && t.Name.Local == "stream" {
				p.depth = 0
				p.rec.discard(p.dec.InputOffset())
				return Event{Kind: Close}, nil
			}
			p.depth--
			if p.depth == 1 {
				end := p.dec.InputOffset()
				raw := p.rec.slice(p.elemStart, end)
				p.rec.discard(end)
				return Event{Kind: Element, Raw: bytes.TrimSpace(raw)}, nil
			}
		default:
			if p.depth == 0 {
				p.rec.discard(p.dec.InputOffset())
			}
		}
	}
}

func isStreamOpen(t xml.StartElement) bool {
	return t.Name.Local == "stream" && (t.Name.Space == StreamNS || t.Name.Space == "stream")
}

func attrMap(t xml.StartElement) map[string]string {
	m := make(map[string]string, len(t.Attr))
	for _, a := range t.Attr {
		key := a.Name.Local
		if a.Name.Space != "" {
			key = a.Name.Space + ":" + a.Name.Local
		}
		m[key] = a.Value
	}
	return m
}

// BuildBody serializes a BOSH <body> wrapper with the given attributes and raw
// child payload. Attributes are emitted in sorted order for determinism.
func BuildBody(attrs map[string]string, payload []byte) []byte {
	var b strings.Builder
	b.WriteString("<body")
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, " %s='%s'", k, escape(attrs[k]))
	}
	// The xmpp: prefix is used by restart/version attributes.
	fmt.Fprintf(&b, " xmlns='%s' xmlns:%s='%s'", NS, XMPPPrefix, XMPPNS)
	if len(payload) == 0 {
		b.WriteString("/>")
		return []byte(b.String())
	}
	b.WriteString(">")
	b.Write(payload)
	b.WriteString("</body>")
	return []byte(b.String())
}

func escape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// Body is a decoded BOSH <body> wrapper.
type Body struct {
	Attrs   map[string]string
	Payload []byte
}

// ParseBody decodes a BOSH <body> wrapper, returning its attributes and the raw
// inner XML (the tunneled stanzas/nonzas).
func ParseBody(data []byte) (*Body, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.RawToken()
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local != "body" {
			return nil, fmt.Errorf("bosh: expected <body>, got <%s>", se.Name.Local)
		}
		attrs := attrMap(se)
		start := dec.InputOffset()
		depth := 1
		end := start
		for depth > 0 {
			off := dec.InputOffset()
			t, terr := dec.RawToken()
			if terr != nil {
				if terr == io.EOF {
					break
				}
				return nil, terr
			}
			switch t.(type) {
			case xml.StartElement:
				depth++
			case xml.EndElement:
				depth--
				if depth == 0 {
					end = off
				}
			}
		}
		if end < start {
			end = start
		}
		payload := bytes.TrimSpace(data[start:end])
		return &Body{Attrs: attrs, Payload: payload}, nil
	}
}

// ClientHeader builds the initiating <stream:stream> header fed into the
// tunneled XMPP stream on session creation or restart.
func ClientHeader(to string) []byte {
	return []byte(fmt.Sprintf(
		"<stream:stream to='%s' xmlns='%s' xmlns:stream='%s' version='1.0'>",
		escape(to), ClientNS, StreamNS))
}

// ServerHeader builds a responding <stream:stream> header the client transport
// synthesizes for its reader from a BOSH create/restart response.
func ServerHeader(from, id string) []byte {
	return []byte(fmt.Sprintf(
		"<stream:stream from='%s' id='%s' xmlns='%s' xmlns:stream='%s' version='1.0'>",
		escape(from), escape(id), ClientNS, StreamNS))
}
