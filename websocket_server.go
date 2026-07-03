package xmpp

import (
	"context"
	"net/http"

	"golang.org/x/net/websocket"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/transport"
)

// WebSocketHandler returns an http.Handler that serves XMPP-over-WebSocket
// (RFC 7395) client connections using the server's negotiation and routing. The
// server must already be initialized (via ListenAndServe or Init).
//
// Mount it on an HTTPS endpoint (path typically /xmpp-websocket):
//
//	http.Handle("/xmpp-websocket", srv.WebSocketHandler())
func (s *Server) WebSocketHandler() http.Handler {
	return websocket.Server{
		Handshake: func(cfg *websocket.Config, _ *http.Request) error {
			// Advertise the RFC 7395 "xmpp" subprotocol.
			cfg.Protocol = []string{"xmpp"}
			return nil
		},
		Handler: func(conn *websocket.Conn) {
			conn.PayloadType = websocket.TextFrame
			trans := transport.NewWebSocket(conn)

			session, err := NewSession(context.Background(), trans,
				WithState(StateServer|StateSecure),
				WithRemoteAddr(jid.JID{}),
			)
			if err != nil {
				_ = conn.Close()
				return
			}
			session.SetFraming(true)

			key := "ws-" + randomID()
			s.mu.Lock()
			s.sessions[key] = session
			s.mu.Unlock()
			defer func() {
				_ = session.Close()
				s.mu.Lock()
				delete(s.sessions, key)
				s.mu.Unlock()
			}()

			s.negotiateAndServe(context.Background(), session)
		},
	}
}
