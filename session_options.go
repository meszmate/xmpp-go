package xmpp

import (
	"github.com/meszmate/xmpp-go/jid"
)

// SessionOption configures a Session.
type SessionOption interface {
	apply(*Session)
}

type sessionOptionFunc func(*Session)

func (f sessionOptionFunc) apply(s *Session) { f(s) }

// WithLocalAddr sets the local JID for the session.
func WithLocalAddr(j jid.JID) SessionOption {
	return sessionOptionFunc(func(s *Session) {
		s.localJID = j
	})
}

// WithRemoteAddr sets the remote JID for the session.
func WithRemoteAddr(j jid.JID) SessionOption {
	return sessionOptionFunc(func(s *Session) {
		s.remoteJID = j
	})
}

// WithState sets the initial session state.
func WithState(state SessionState) SessionOption {
	return sessionOptionFunc(func(s *Session) {
		s.state.Store(uint32(state))
	})
}

// WithMux sets the stanza multiplexer.
func WithMux(mux *Mux) SessionOption {
	return sessionOptionFunc(func(s *Session) {
		s.mux = mux
	})
}
