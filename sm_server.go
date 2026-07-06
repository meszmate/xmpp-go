package xmpp

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/meszmate/xmpp-go/internal/ns"
)

// detachedEntry is a dropped-but-resumable session parked awaiting a <resume/>.
type detachedEntry struct {
	holder *Session
	timer  *time.Timer
}

// serverEnableSM handles a client <enable/> request, activating Stream
// Management for the (bound) session and, when resume='true' is requested,
// issuing a resumption id the client can later present via <resume/>.
func (s *Server) serverEnableSM(ctx context.Context, session *Session, start *xml.StartElement) error {
	if err := session.Reader().Skip(); err != nil {
		return err
	}
	if session.State()&StateReady == 0 {
		// Stream Management may only be enabled after resource binding.
		return session.SendRaw(ctx, strings.NewReader(
			fmt.Sprintf("<failed xmlns='%s'><unexpected-request xmlns='urn:ietf:params:xml:ns:xmpp-stanzas'/></failed>", ns.SM)))
	}

	session.EnableSM()
	if attrTrue(start.Attr, "resume") {
		previd := randomID()
		maxSecs := int(s.resumeWindow().Seconds())
		session.SetSMResume(previd, maxSecs)
		return session.SendRaw(ctx, strings.NewReader(
			fmt.Sprintf("<enabled xmlns='%s' id='%s' resume='true' max='%d'/>", ns.SM, previd, maxSecs)))
	}
	return session.SendRaw(ctx, strings.NewReader("<enabled xmlns='"+ns.SM+"'/>"))
}

// serverResume handles a client <resume/> on a freshly-authenticated stream. It
// looks up the parked session by previd, verifies it belongs to the same
// authenticated identity, re-attaches its Stream Management state to this new
// session, acknowledges, and replays any unacknowledged stanzas.
func (s *Server) serverResume(ctx context.Context, session *Session, start *xml.StartElement) error {
	if err := session.Reader().Skip(); err != nil {
		return err
	}
	if session.State()&StateAuthenticated == 0 {
		return s.smFailed(ctx, session, "not-authorized")
	}
	previd := attrValue(start.Attr, "previd")
	clientH := parseUintAttr(start.Attr, "h")

	s.mu.Lock()
	entry := s.detached[previd]
	if entry != nil {
		delete(s.detached, previd)
	}
	s.mu.Unlock()
	if entry == nil {
		return s.smFailed(ctx, session, "item-not-found")
	}
	if entry.timer != nil {
		entry.timer.Stop()
	}
	holder := entry.holder

	// The resumed session must belong to the same account that authenticated.
	if !holder.RemoteAddr().Bare().Equal(session.RemoteAddr().Bare()) {
		// Re-park so a legitimate owner can still resume within the window.
		s.parkHolder(previd, holder)
		return s.smFailed(ctx, session, "not-authorized")
	}

	session.SetRemoteAddr(holder.RemoteAddr())
	session.SetState(StateBound | StateReady)

	// Atomically adopt the parked SM state and install successor forwarding so no
	// concurrently-delivered stanza is lost, then re-point the router, ack, and
	// replay — all under session.mu so live deliveries queue behind the replay.
	session.mu.Lock()

	holder.smMu.Lock()
	session.smMu.Lock()
	session.smOutbound = holder.smOutbound
	session.smQueue = append([]smEntry(nil), holder.smQueue...)
	session.smPrevID = holder.smPrevID
	session.smMax = holder.smMax
	session.smMu.Unlock()
	session.smInbound.Store(holder.smInbound.Load())
	session.smEnabled.Store(true)
	holder.smSuccessor = session
	holder.smMu.Unlock()

	s.router.register(holder.RemoteAddr(), session)

	reply := fmt.Sprintf("<resumed xmlns='%s' previd='%s' h='%d'/>", ns.SM, previd, session.SMHandled())
	if _, err := session.writer.WriteRaw([]byte(reply)); err != nil {
		session.mu.Unlock()
		return err
	}
	replayErr := session.replayUnacked(clientH)
	session.mu.Unlock()
	if replayErr != nil {
		return replayErr
	}

	// The resumed session gets a fresh set of per-session plugins (bind — and its
	// plugin activation — was skipped on this stream).
	s.initSessionPlugins(ctx, session)
	return nil
}

// numDetached returns the number of sessions currently parked awaiting resume.
func (s *Server) numDetached() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.detached)
}

// smFailed reports a Stream Management failure with the given stanza condition.
func (s *Server) smFailed(ctx context.Context, session *Session, condition string) error {
	payload := fmt.Sprintf("<failed xmlns='%s'><%s xmlns='urn:ietf:params:xml:ns:xmpp-stanzas'/></failed>", ns.SM, condition)
	return session.SendRaw(ctx, strings.NewReader(payload))
}

// parkForResume parks a dropped but SM-resumable session so its state survives
// for the resumption window. It returns true when the session was parked (and
// therefore must not be torn down by the caller).
func (s *Server) parkForResume(old *Session) bool {
	previd := old.SMPrevID()
	if previd == "" || old.State()&StateReady == 0 {
		return false
	}
	// Never re-park a session that was already resumed onto a successor.
	old.smMu.Lock()
	resumed := old.smSuccessor != nil
	old.smMu.Unlock()
	if resumed {
		return false
	}

	holder := s.newHolder(old)
	// Route deliveries during the detached window into the holder's buffer.
	s.router.register(old.RemoteAddr(), holder)
	// The old session's plugins are bound to its now-dead transport.
	s.closeSessionPlugins(old)
	s.parkHolder(previd, holder)
	return true
}

// resumeWindow returns how long dropped sessions are held for resumption.
func (s *Server) resumeWindow() time.Duration {
	if s.opts.resumeTimeout > 0 {
		return s.opts.resumeTimeout
	}
	return smResumeTimeout * time.Second
}

// parkHolder registers a holder session under previd and arms its expiry timer.
func (s *Server) parkHolder(previd string, holder *Session) {
	entry := &detachedEntry{holder: holder}
	entry.timer = time.AfterFunc(s.resumeWindow(), func() {
		s.expireDetached(previd, holder)
	})
	s.mu.Lock()
	s.detached[previd] = entry
	s.mu.Unlock()
}

// newHolder creates a lightweight session that carries a dropped session's
// identity and Stream Management state while it is parked, buffering any
// delivered stanzas for replay on resume.
func (s *Server) newHolder(old *Session) *Session {
	holder := &Session{
		closed: make(chan struct{}),
		smAck:  make(chan uint32, 1),
		mux:    NewMux(),
	}
	holder.SetRemoteAddr(old.RemoteAddr())
	holder.SetState(old.State())
	holder.adoptSMState(old)
	holder.setDetached(true)
	return holder
}

// expireDetached tears down a parked session whose resumption window elapsed
// without a successful resume. It only acts if the holder is still the parked
// entry (guarding against a race with a concurrent resume).
func (s *Server) expireDetached(previd string, holder *Session) {
	s.mu.Lock()
	entry := s.detached[previd]
	if entry == nil || entry.holder != holder {
		s.mu.Unlock()
		return
	}
	delete(s.detached, previd)
	s.mu.Unlock()

	s.router.unregisterIf(holder.RemoteAddr(), holder)
	close(holder.closed)
}
