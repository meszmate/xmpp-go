package xmpp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/transport"
)

// S2SResolver maps a remote XMPP domain to a dialable "host:port" address. It
// replaces DNS SRV resolution of `_xmpp-server._tcp` and is required for
// server-to-server federation (also handy for testing and private deployments).
type S2SResolver func(domain string) (addr string, ok bool)

// dialbackKey computes the XEP-0185 server-dialback key binding the receiving
// domain, originating domain, and stream id under the originating server's
// secret. Only the originating (authoritative) server knows the secret, so only
// it can generate and later verify a given key.
func dialbackKey(secret, receiving, originating, streamID string) string {
	h := sha256.Sum256([]byte(secret))
	mac := hmac.New(sha256.New, []byte(hex.EncodeToString(h[:])))
	fmt.Fprintf(mac, "%s %s %s", receiving, originating, streamID)
	return hex.EncodeToString(mac.Sum(nil))
}

// s2sEnabled reports whether server-to-server federation is configured.
func (s *Server) s2sEnabled() bool { return s.opts.s2sResolver != nil }

// isLocalDomain reports whether a JID's domain is served by this server.
func (s *Server) isLocalDomain(j jid.JID) bool {
	return strings.EqualFold(j.Domain(), s.domain)
}

// --- outbound (originating server) ---

// s2sOutbound is a shared outbound s2s stream to a remote domain.
type s2sOutbound struct {
	remote  string
	session *Session
	ready   chan struct{}
	err     error
}

// ensureS2SOut returns the (possibly still-connecting) outbound stream to remote,
// creating one if none exists.
func (s *Server) ensureS2SOut(remote string) *s2sOutbound {
	s.s2sMu.Lock()
	if s.s2sOut == nil {
		s.s2sOut = make(map[string]*s2sOutbound)
	}
	if out := s.s2sOut[remote]; out != nil {
		s.s2sMu.Unlock()
		return out
	}
	out := &s2sOutbound{remote: remote, ready: make(chan struct{})}
	s.s2sOut[remote] = out
	s.s2sMu.Unlock()

	go s.establishS2SOut(out)
	return out
}

// establishS2SOut dials the remote server and completes dialback authentication,
// then starts serving inbound stanzas on the stream.
func (s *Server) establishS2SOut(out *s2sOutbound) {
	defer close(out.ready)

	addr, ok := s.opts.s2sResolver(out.remote)
	if !ok {
		out.err = fmt.Errorf("xmpp: no s2s route for %q", out.remote)
		s.forgetS2SOut(out.remote)
		return
	}
	conn, err := (&net.Dialer{Timeout: 10 * time.Second}).Dial("tcp", addr)
	if err != nil {
		out.err = err
		s.forgetS2SOut(out.remote)
		return
	}
	session, err := NewSession(context.Background(), transport.NewTCP(conn), WithState(StateServer|StateS2S))
	if err != nil {
		_ = conn.Close()
		out.err = err
		s.forgetS2SOut(out.remote)
		return
	}

	if err := s.s2sDialbackOut(session, out.remote); err != nil {
		_ = session.Close()
		out.err = err
		s.forgetS2SOut(out.remote)
		return
	}
	out.session = session
	go s.s2sInboundStanzas(session, out.remote, map[string]bool{out.remote: true})
}

func (s *Server) forgetS2SOut(remote string) {
	s.s2sMu.Lock()
	delete(s.s2sOut, remote)
	s.s2sMu.Unlock()
}

// s2sDialbackOut runs the originating-server side of dialback on an outbound
// stream: open the stream, learn the receiving server's stream id, send a
// db:result key, and await the receiving server's valid verdict.
func (s *Server) s2sDialbackOut(session *Session, remote string) error {
	if err := writeS2SStreamHeader(session, s.domain, remote, ""); err != nil {
		return err
	}
	rsID, err := readS2SStreamID(session)
	if err != nil {
		return err
	}

	key := dialbackKey(s.opts.s2sSecret, remote, s.domain, rsID)
	result := fmt.Sprintf("<db:result xmlns:db='%s' from='%s' to='%s'>%s</db:result>",
		ns.Dialback, s.domain, remote, key)
	if err := session.SendRaw(context.Background(), strings.NewReader(result)); err != nil {
		return err
	}

	// Await <db:result type='valid'/> (skipping features and any other traffic).
	reader := session.Reader()
	for {
		tok, err := reader.Token()
		if err != nil {
			return err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local == "result" && (se.Name.Space == ns.Dialback || attrValue(se.Attr, "xmlns:db") != "") {
			typ := attrValue(se.Attr, "type")
			_ = reader.Skip()
			if typ == "valid" {
				return nil
			}
			return errors.New("xmpp: s2s dialback rejected by " + remote)
		}
		if err := reader.Skip(); err != nil {
			return err
		}
	}
}

// deliverRemote routes a stanza to a remote domain over s2s, establishing the
// outbound stream on demand.
func (s *Server) deliverRemote(ctx context.Context, st stanza.Stanza, to jid.JID) {
	out := s.ensureS2SOut(to.Domain())
	select {
	case <-out.ready:
	case <-ctx.Done():
		return
	case <-time.After(15 * time.Second):
		return
	}
	if out.err != nil || out.session == nil {
		return
	}
	_ = out.session.Send(ctx, st)
}

// --- inbound (receiving / authoritative server) ---

// serveS2SInbound handles an inbound server-to-server stream after its opening
// <stream:stream> header (streamStart) has been read. It performs the receiving-
// and authoritative-server roles of dialback and routes authorized stanzas.
func (s *Server) serveS2SInbound(ctx context.Context, session *Session, streamStart *xml.StartElement) error {
	remoteFrom := attrValue(streamStart.Attr, "from")
	streamID := randomID()
	if err := writeS2SStreamHeader(session, s.domain, remoteFrom, streamID); err != nil {
		return err
	}
	// Advertise (empty) features so a peer that waits for them can proceed.
	if err := session.SendRaw(ctx, strings.NewReader(
		fmt.Sprintf("<stream:features xmlns:stream='%s'><dialback xmlns='urn:xmpp:features:dialback'/></stream:features>", ns.Stream))); err != nil {
		return err
	}

	authorized := map[string]bool{}
	reader := session.Reader()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		tok, err := reader.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch {
		case start.Name.Local == "verify":
			if err := s.s2sHandleVerify(ctx, session, &start); err != nil {
				return err
			}
		case start.Name.Local == "result":
			if err := s.s2sHandleResult(ctx, session, &start, streamID, authorized); err != nil {
				return err
			}
		case start.Name.Local == "message", start.Name.Local == "presence", start.Name.Local == "iq":
			if err := s.s2sRouteInbound(ctx, session, &start, authorized); err != nil {
				return err
			}
		default:
			if err := reader.Skip(); err != nil {
				return err
			}
		}
	}
}

// s2sHandleVerify answers a dialback verification request as the authoritative
// server: recompute the key under our secret and report valid/invalid.
func (s *Server) s2sHandleVerify(ctx context.Context, session *Session, start *xml.StartElement) error {
	from := attrValue(start.Attr, "from") // receiving server (RS)
	to := attrValue(start.Attr, "to")     // us (authoritative)
	id := attrValue(start.Attr, "id")
	key, err := readElementText(session, start)
	if err != nil {
		return err
	}

	verdict := "invalid"
	if strings.EqualFold(to, s.domain) {
		expected := dialbackKey(s.opts.s2sSecret, from, to, id)
		if hmac.Equal([]byte(expected), []byte(strings.TrimSpace(key))) {
			verdict = "valid"
		}
	}
	reply := fmt.Sprintf("<db:verify xmlns:db='%s' from='%s' to='%s' id='%s' type='%s'/>",
		ns.Dialback, s.domain, from, id, verdict)
	return session.SendRaw(ctx, strings.NewReader(reply))
}

// s2sHandleResult processes a db:result from an originating server: verify its
// key by connecting back to the originating (authoritative) server, then accept
// or reject the stream for that domain.
func (s *Server) s2sHandleResult(ctx context.Context, session *Session, start *xml.StartElement, streamID string, authorized map[string]bool) error {
	from := attrValue(start.Attr, "from") // originating server (OS)
	to := attrValue(start.Attr, "to")     // us (RS)
	key, err := readElementText(session, start)
	if err != nil {
		return err
	}
	if !strings.EqualFold(to, s.domain) {
		return s.sendDialbackResult(ctx, session, from, "invalid")
	}

	valid := s.s2sVerifyRemote(from, streamID, strings.TrimSpace(key))
	verdict := "invalid"
	if valid {
		verdict = "valid"
		authorized[strings.ToLower(from)] = true
	}
	return s.sendDialbackResult(ctx, session, from, verdict)
}

func (s *Server) sendDialbackResult(ctx context.Context, session *Session, to, verdict string) error {
	reply := fmt.Sprintf("<db:result xmlns:db='%s' from='%s' to='%s' type='%s'/>",
		ns.Dialback, s.domain, to, verdict)
	return session.SendRaw(ctx, strings.NewReader(reply))
}

// s2sVerifyRemote connects to the originating domain's authoritative server and
// asks it to confirm the dialback key (the receiving-server role of dialback).
func (s *Server) s2sVerifyRemote(originating, streamID, key string) bool {
	addr, ok := s.opts.s2sResolver(originating)
	if !ok {
		return false
	}
	conn, err := (&net.Dialer{Timeout: 10 * time.Second}).Dial("tcp", addr)
	if err != nil {
		return false
	}
	session, err := NewSession(context.Background(), transport.NewTCP(conn), WithState(StateServer|StateS2S))
	if err != nil {
		_ = conn.Close()
		return false
	}
	defer session.Close()

	if err := writeS2SStreamHeader(session, s.domain, originating, ""); err != nil {
		return false
	}
	if _, err := readS2SStreamID(session); err != nil {
		return false
	}
	verify := fmt.Sprintf("<db:verify xmlns:db='%s' from='%s' to='%s' id='%s'>%s</db:verify>",
		ns.Dialback, s.domain, originating, streamID, key)
	if err := session.SendRaw(context.Background(), strings.NewReader(verify)); err != nil {
		return false
	}

	reader := session.Reader()
	deadline := time.Now().Add(10 * time.Second)
	if d, ok := session.Transport().(interface{ SetDeadline(time.Time) error }); ok {
		_ = d.SetDeadline(deadline)
	}
	for {
		tok, err := reader.Token()
		if err != nil {
			return false
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local == "verify" {
			typ := attrValue(se.Attr, "type")
			_ = reader.Skip()
			return typ == "valid"
		}
		if err := reader.Skip(); err != nil {
			return false
		}
	}
}

// s2sRouteInbound decodes and routes a stanza received over an authorized s2s
// stream to a local recipient, enforcing that its from-domain is authorized.
func (s *Server) s2sRouteInbound(ctx context.Context, session *Session, start *xml.StartElement, authorized map[string]bool) error {
	switch start.Name.Local {
	case "message":
		var msg stanza.Message
		if err := session.Reader().DecodeElement(&msg, start); err != nil {
			return err
		}
		if !s.s2sFromAuthorized(msg.From, authorized) || !s.isLocalDomain(msg.To) {
			return nil
		}
		for _, dst := range s.router.targets(msg.To) {
			_ = dst.Send(ctx, &msg)
		}
	case "presence":
		var pres stanza.Presence
		if err := session.Reader().DecodeElement(&pres, start); err != nil {
			return err
		}
		if !s.s2sFromAuthorized(pres.From, authorized) {
			return nil
		}
		for _, dst := range s.router.targets(pres.To) {
			_ = dst.Send(ctx, &pres)
		}
	case "iq":
		var iq stanza.IQ
		if err := session.Reader().DecodeElement(&iq, start); err != nil {
			return err
		}
		if !s.s2sFromAuthorized(iq.From, authorized) || !s.isLocalDomain(iq.To) {
			return nil
		}
		for _, dst := range s.router.targets(iq.To) {
			_ = dst.Send(ctx, &iq)
		}
	}
	return nil
}

func (s *Server) s2sFromAuthorized(from jid.JID, authorized map[string]bool) bool {
	return !from.IsZero() && authorized[strings.ToLower(from.Domain())]
}

// s2sInboundStanzas serves stanzas arriving on an outbound stream (the remote
// server may push stanzas back to us over it), routing them to local users.
func (s *Server) s2sInboundStanzas(session *Session, remote string, authorized map[string]bool) {
	reader := session.Reader()
	for {
		tok, err := reader.Token()
		if err != nil {
			return
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "message", "presence", "iq":
			_ = s.s2sRouteInbound(context.Background(), session, &start, authorized)
		default:
			if err := reader.Skip(); err != nil {
				return
			}
		}
	}
}

// --- shared helpers ---

// writeS2SStreamHeader writes a jabber:server stream header with the dialback
// namespace declared.
func writeS2SStreamHeader(session *Session, from, to, id string) error {
	var b strings.Builder
	b.WriteString("<?xml version='1.0'?><stream:stream xmlns='")
	b.WriteString(ns.Server)
	b.WriteString("' xmlns:stream='")
	b.WriteString(ns.Stream)
	b.WriteString("' xmlns:db='")
	b.WriteString(ns.Dialback)
	b.WriteString("'")
	if from != "" {
		fmt.Fprintf(&b, " from='%s'", from)
	}
	if to != "" {
		fmt.Fprintf(&b, " to='%s'", to)
	}
	if id != "" {
		fmt.Fprintf(&b, " id='%s'", id)
	}
	b.WriteString(" version='1.0'>")
	return session.SendRaw(context.Background(), strings.NewReader(b.String()))
}

// readS2SStreamID reads tokens until the peer's <stream:stream> header and
// returns its id attribute.
func readS2SStreamID(session *Session) (string, error) {
	reader := session.Reader()
	for {
		tok, err := reader.Token()
		if err != nil {
			return "", err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if isStreamNS(se.Name) && se.Name.Local == "stream" {
			return attrValue(se.Attr, "id"), nil
		}
	}
}

// readElementText decodes an element's text content given its start token.
func readElementText(session *Session, start *xml.StartElement) (string, error) {
	var v struct {
		Text string `xml:",chardata"`
	}
	if err := session.Reader().DecodeElement(&v, start); err != nil {
		return "", err
	}
	return v.Text, nil
}
