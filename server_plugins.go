package xmpp

import (
	"bytes"
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/stanza"
)

// initSessionPlugins builds and initializes a per-session plugin.Manager from
// the configured factory, wiring session-scoped hooks so plugins can send on and
// register inbound handlers for this session. It is a no-op when no factory is
// configured. Called once per session, after resource binding succeeds.
func (s *Server) initSessionPlugins(ctx context.Context, session *Session) {
	if s.opts.pluginFactory == nil {
		return
	}
	plugins := s.opts.pluginFactory()
	if len(plugins) == 0 {
		return
	}

	mgr := plugin.NewManager()
	for _, p := range plugins {
		if err := mgr.Register(p); err != nil {
			_ = mgr.Close()
			return
		}
	}

	params := plugin.InitParams{
		SendRaw: func(ctx context.Context, data []byte) error {
			return session.SendRaw(ctx, bytes.NewReader(data))
		},
		SendElement: session.SendElement,
		State:       func() uint32 { return uint32(session.State()) },
		LocalJID:    func() string { return s.domain },
		RemoteJID:   func() string { return session.RemoteAddr().String() },
		Storage:     s.opts.storage,
		Get:         mgr.Get,
		Handle: func(name xml.Name, stanzaType string, h plugin.IncomingHandler) {
			session.Mux().Handle(name, stanzaType, HandlerFunc(func(ctx context.Context, _ *Session, st stanza.Stanza) error {
				return h(ctx, st)
			}))
		},
		// Request/response is not available on server sessions: inbound stanzas
		// are consumed by the synchronous negotiation loop, so there is no
		// correlation waiter. Plugins needing it must run client-side.
	}
	if err := mgr.Initialize(ctx, params); err != nil {
		_ = mgr.Close()
		return
	}

	s.mu.Lock()
	s.sessionPlugins[session] = mgr
	s.mu.Unlock()
}

// closeSessionPlugins tears down a session's plugin manager, if any.
func (s *Server) closeSessionPlugins(session *Session) {
	s.mu.Lock()
	mgr := s.sessionPlugins[session]
	delete(s.sessionPlugins, session)
	s.mu.Unlock()
	if mgr != nil {
		_ = mgr.Close()
	}
}

// dispatchSessionPlugin offers an inbound stanza to the session's plugin routes,
// reporting whether a plugin handled it.
func (s *Server) dispatchSessionPlugin(ctx context.Context, session *Session, st stanza.Stanza) (bool, error) {
	s.mu.Lock()
	mgr := s.sessionPlugins[session]
	s.mu.Unlock()
	if mgr == nil {
		return false, nil
	}
	return session.Mux().dispatch(ctx, session, st)
}
