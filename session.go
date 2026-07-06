package xmpp

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/transport"
	xmppxml "github.com/meszmate/xmpp-go/xml"
)

// SessionState represents the state of an XMPP session.
type SessionState uint32

const (
	StateSecure        SessionState = 1 << iota // TLS negotiated
	StateAuthenticated                          // SASL complete
	StateBound                                  // Resource bound
	StateReady                                  // Fully negotiated
	StateServer                                 // Server role
	StateS2S                                    // Server-to-server
)

// Session represents an XMPP session (client or server).
type Session struct {
	state     atomic.Uint32
	mu        sync.Mutex
	trans     transport.Transport
	localJID  jid.JID
	remoteJID jid.JID
	reader    *xmppxml.StreamReader
	writer    *xmppxml.StreamWriter
	mux       *Mux
	closed    chan struct{}
	err       error

	// Stream Management (XEP-0198) state.
	smEnabled atomic.Bool
	smInbound atomic.Uint32 // count of handled inbound stanzas (our "h")
	smAck     chan uint32   // delivers received <a h='N'/> values to a waiter

	// Stream Management resumption (XEP-0198 §5) state, guarded by smMu.
	smMu        sync.Mutex
	smOutbound  uint32    // count of stanzas sent under SM (their seq numbers)
	smQueue     []smEntry // unacknowledged outbound stanzas, for replay on resume
	smDetached  bool      // server: parked awaiting resume; Send buffers instead of writing
	smPrevID    string    // resumption id ("previd")
	smMax       int       // negotiated resumption timeout, seconds
	smSuccessor *Session  // server: session that adopted this one on resume (forward sends)

	// framing selects RFC 7395 <open/>-framed streams (WebSocket) instead of
	// <stream:stream>. Set before negotiation begins.
	framing bool

	// saslMech records the SASL mechanism that authenticated the session.
	saslMech string
}

// SetSASLMechanism records the SASL mechanism used to authenticate.
func (s *Session) SetSASLMechanism(name string) { s.saslMech = name }

// SASLMechanism returns the SASL mechanism that authenticated the session, or ""
// if unauthenticated.
func (s *Session) SASLMechanism() string { return s.saslMech }

// smEntry is a queued outbound stanza awaiting acknowledgement, tagged with the
// Stream Management sequence number in effect when it was sent.
type smEntry struct {
	seq  uint32
	elem any
}

// NewSession creates a new XMPP session with the given transport and options.
func NewSession(ctx context.Context, trans transport.Transport, opts ...SessionOption) (*Session, error) {
	s := &Session{
		trans:  trans,
		reader: xmppxml.NewStreamReader(trans),
		writer: xmppxml.NewStreamWriter(trans),
		mux:    NewMux(),
		closed: make(chan struct{}),
		smAck:  make(chan uint32, 1),
	}

	for _, opt := range opts {
		opt.apply(s)
	}

	return s, nil
}

// Send sends a stanza through the session.
func (s *Session) Send(ctx context.Context, st stanza.Stanza) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closed:
		return errors.New("xmpp: session closed")
	default:
	}

	if s.smEnabled.Load() {
		return s.smSend(st)
	}
	return s.writer.Encode(st)
}

// smSend queues an outbound stanza for Stream Management (so it can be replayed
// after a resumed connection) and writes it — unless the session is detached
// (server-side, awaiting resume), in which case it is only buffered. Callers
// must hold s.mu.
func (s *Session) smSend(elem any) error {
	s.smMu.Lock()
	// If this session has been resumed onto a successor, forward the stanza there
	// so late deliveries during handoff are not lost.
	if s.smSuccessor != nil {
		succ := s.smSuccessor
		s.smMu.Unlock()
		return succ.SendElement(context.Background(), elem)
	}
	s.smOutbound++
	s.smQueue = append(s.smQueue, smEntry{seq: s.smOutbound, elem: elem})
	detached := s.smDetached
	s.smMu.Unlock()

	if detached {
		// No live transport; the stanza is buffered for replay on resume.
		return nil
	}
	return s.writer.Encode(elem)
}

// SendRaw writes raw XML to the stream.
func (s *Session) SendRaw(ctx context.Context, r io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closed:
		return errors.New("xmpp: session closed")
	default:
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	_, err = s.writer.WriteRaw(data)
	return err
}

// SendElement encodes an XML element to the stream.
func (s *Session) SendElement(ctx context.Context, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closed:
		return errors.New("xmpp: session closed")
	default:
	}

	if s.smEnabled.Load() {
		return s.smSend(v)
	}
	return s.writer.Encode(v)
}

// Serve reads stanzas from the stream and dispatches them to the mux.
func (s *Session) Serve(handler Handler) error {
	if handler == nil {
		handler = s.mux
	}
	for {
		select {
		case <-s.closed:
			return s.err
		default:
		}

		tok, err := s.reader.Token()
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

		// Stream Management (XEP-0198) control elements.
		if start.Name.Space == ns.SM {
			if err := s.handleSMControl(start); err != nil {
				return err
			}
			continue
		}

		var st stanza.Stanza
		switch start.Name.Local {
		case "message":
			msg := &stanza.Message{}
			if err := s.reader.DecodeElement(msg, &start); err != nil {
				return err
			}
			st = msg
		case "presence":
			pres := &stanza.Presence{}
			if err := s.reader.DecodeElement(pres, &start); err != nil {
				return err
			}
			st = pres
		case "iq":
			iq := &stanza.IQ{}
			if err := s.reader.DecodeElement(iq, &start); err != nil {
				return err
			}
			st = iq
		default:
			if err := s.reader.Skip(); err != nil {
				return err
			}
			continue
		}

		s.smCountInbound()

		// A handler error or panic on a single incoming stanza must not tear
		// down the whole receive loop (which would silently stop all further
		// message delivery). Isolate it and continue.
		dispatchStanza(handler, s, st)
	}
}

// SetFraming selects RFC 7395 <open/> stream framing (used by WebSocket).
func (s *Session) SetFraming(v bool) { s.framing = v }

// Framing reports whether the session uses RFC 7395 <open/> framing.
func (s *Session) Framing() bool { return s.framing }

// EnableSM marks Stream Management (XEP-0198) as active on the session.
func (s *Session) EnableSM() { s.smEnabled.Store(true) }

// SMEnabled reports whether Stream Management is active.
func (s *Session) SMEnabled() bool { return s.smEnabled.Load() }

// SMHandled returns the number of inbound stanzas handled ("h") since SM was
// enabled.
func (s *Session) SMHandled() uint32 { return s.smInbound.Load() }

func (s *Session) smCountInbound() {
	if s.smEnabled.Load() {
		s.smInbound.Add(1)
	}
}

// handleSMControl processes an XEP-0198 control element (r/a/enable/enabled),
// consuming it from the stream.
func (s *Session) handleSMControl(start xml.StartElement) error {
	if err := s.reader.Skip(); err != nil {
		return err
	}
	switch start.Name.Local {
	case "r":
		return s.sendSMAck()
	case "a":
		h := parseSMHAttr(start.Attr)
		// The peer acknowledged h of our outbound stanzas: drop them from the
		// replay queue.
		s.smTrimOutbound(h)
		select {
		case s.smAck <- h:
		default:
		}
	case "enable":
		s.EnableSM()
		return s.writeSMRaw("<enabled xmlns='" + ns.SM + "'/>")
	}
	return nil
}

// smTrimOutbound drops acknowledged stanzas (seq <= h) from the replay queue.
func (s *Session) smTrimOutbound(h uint32) {
	s.smMu.Lock()
	defer s.smMu.Unlock()
	i := 0
	for i < len(s.smQueue) && s.smQueue[i].seq <= h {
		i++
	}
	if i > 0 {
		s.smQueue = append(s.smQueue[:0], s.smQueue[i:]...)
	}
}

// resetSM clears all Stream Management state, used when a resume attempt is
// declined and the session falls back to a fresh resource bind.
func (s *Session) resetSM() {
	s.smEnabled.Store(false)
	s.smInbound.Store(0)
	s.smMu.Lock()
	s.smOutbound = 0
	s.smQueue = nil
	s.smPrevID = ""
	s.smMax = 0
	s.smMu.Unlock()
}

// SetSMResume records the resumption id and negotiated timeout for the session.
func (s *Session) SetSMResume(previd string, max int) {
	s.smMu.Lock()
	defer s.smMu.Unlock()
	s.smPrevID = previd
	s.smMax = max
}

// SMPrevID returns the resumption id, or "" if resumption is not available.
func (s *Session) SMPrevID() string {
	s.smMu.Lock()
	defer s.smMu.Unlock()
	return s.smPrevID
}

// SMResumable reports whether the session negotiated resumption support.
func (s *Session) SMResumable() bool {
	return s.SMPrevID() != ""
}

// smOutboundCount returns the number of stanzas sent under SM.
func (s *Session) smOutboundCount() uint32 {
	s.smMu.Lock()
	defer s.smMu.Unlock()
	return s.smOutbound
}

// smQueueSnapshot returns a copy of the current unacknowledged outbound queue.
func (s *Session) smQueueSnapshot() []smEntry {
	s.smMu.Lock()
	defer s.smMu.Unlock()
	out := make([]smEntry, len(s.smQueue))
	copy(out, s.smQueue)
	return out
}

// setDetached marks the session as parked (server-side) so Send buffers stanzas
// for replay instead of writing to a dead transport.
func (s *Session) setDetached(v bool) {
	s.smMu.Lock()
	defer s.smMu.Unlock()
	s.smDetached = v
}

// isDetached reports whether the session is parked awaiting resume.
func (s *Session) isDetached() bool {
	s.smMu.Lock()
	defer s.smMu.Unlock()
	return s.smDetached
}

// adoptSMState transfers Stream Management counters, queue, and resumption id
// from a prior (dropped) session onto this one when resuming. Called on the new
// session before it begins serving.
func (s *Session) adoptSMState(prev *Session) {
	prev.smMu.Lock()
	outbound := prev.smOutbound
	queue := make([]smEntry, len(prev.smQueue))
	copy(queue, prev.smQueue)
	previd := prev.smPrevID
	max := prev.smMax
	prev.smMu.Unlock()

	inbound := prev.smInbound.Load()

	s.smMu.Lock()
	s.smOutbound = outbound
	s.smQueue = queue
	s.smPrevID = previd
	s.smMax = max
	s.smMu.Unlock()

	s.smInbound.Store(inbound)
	s.EnableSM()
}

// replayUnacked resends every queued stanza with seq greater than h (those the
// peer has not acknowledged), in order. Callers must hold s.mu; used during
// resumption after the peer reports its handled count.
func (s *Session) replayUnacked(h uint32) error {
	s.smMu.Lock()
	// Drop anything the peer already acknowledged, then snapshot the rest.
	i := 0
	for i < len(s.smQueue) && s.smQueue[i].seq <= h {
		i++
	}
	if i > 0 {
		s.smQueue = append(s.smQueue[:0], s.smQueue[i:]...)
	}
	pending := make([]smEntry, len(s.smQueue))
	copy(pending, s.smQueue)
	s.smMu.Unlock()

	for _, e := range pending {
		if err := s.writer.Encode(e.elem); err != nil {
			return err
		}
	}
	return nil
}

// sendSMAck writes an <a h='N'/> acknowledging the stanzas handled so far.
func (s *Session) sendSMAck() error {
	return s.writeSMRaw(fmt.Sprintf("<a xmlns='%s' h='%d'/>", ns.SM, s.smInbound.Load()))
}

// RequestSMAck sends an <r/> and waits for the peer's <a h='N'/>, returning the
// number of this session's outbound stanzas the peer reports as handled. It
// requires SM to be enabled.
func (s *Session) RequestSMAck(ctx context.Context) (uint32, error) {
	if !s.SMEnabled() {
		return 0, errors.New("xmpp: stream management not enabled")
	}
	// Drain any stale ack.
	select {
	case <-s.smAck:
	default:
	}
	if err := s.writeSMRaw("<r xmlns='" + ns.SM + "'/>"); err != nil {
		return 0, err
	}
	select {
	case h := <-s.smAck:
		return h, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-s.closed:
		return 0, errors.New("xmpp: session closed")
	}
}

func (s *Session) writeSMRaw(payload string) error {
	return s.SendRaw(context.Background(), strings.NewReader(payload))
}

func parseSMHAttr(attrs []xml.Attr) uint32 {
	for _, a := range attrs {
		if a.Name.Local == "h" {
			var n uint64
			for _, c := range a.Value {
				if c < '0' || c > '9' {
					break
				}
				n = n*10 + uint64(c-'0')
			}
			return uint32(n)
		}
	}
	return 0
}

// dispatchStanza invokes the handler for one stanza, recovering from panics and
// swallowing per-stanza handler errors so the receive loop survives.
func dispatchStanza(handler Handler, s *Session, st stanza.Stanza) {
	defer func() { _ = recover() }()
	_ = handler.HandleStanza(context.Background(), s, st)
}

// Close closes the session.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.closed:
		return nil
	default:
		close(s.closed)
	}

	return s.trans.Close()
}

// Restart resets the stream reader and writer over the current transport.
//
// It must be called after a stream-altering negotiation step (STARTTLS or SASL
// success) so that a fresh XML stream — with a new root <stream:stream> — is
// parsed and written over the possibly-upgraded transport. Callers must ensure
// no concurrent Send/Serve is in flight; during negotiation this is guaranteed
// because the receive loop has not yet started.
func (s *Session) Restart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reader = xmppxml.NewStreamReader(s.trans)
	s.writer = xmppxml.NewStreamWriter(s.trans)
}

// State returns the current session state.
func (s *Session) State() SessionState {
	return SessionState(s.state.Load())
}

// SetState sets session state flags.
func (s *Session) SetState(state SessionState) {
	for {
		cur := s.state.Load()
		next := cur | uint32(state)
		if s.state.CompareAndSwap(cur, next) {
			return
		}
	}
}

// LocalAddr returns the local JID.
func (s *Session) LocalAddr() jid.JID {
	return s.localJID
}

// RemoteAddr returns the remote JID.
func (s *Session) RemoteAddr() jid.JID {
	return s.remoteJID
}

// SetLocalAddr sets the local JID.
func (s *Session) SetLocalAddr(j jid.JID) {
	s.localJID = j
}

// SetRemoteAddr sets the remote JID.
func (s *Session) SetRemoteAddr(j jid.JID) {
	s.remoteJID = j
}

// Transport returns the underlying transport.
func (s *Session) Transport() transport.Transport {
	return s.trans
}

// Reader returns the XML stream reader.
func (s *Session) Reader() *xmppxml.StreamReader {
	return s.reader
}

// Writer returns the XML stream writer.
func (s *Session) Writer() *xmppxml.StreamWriter {
	return s.writer
}

// Mux returns the stanza multiplexer.
func (s *Session) Mux() *Mux {
	return s.mux
}
