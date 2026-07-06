package transport

import (
	"crypto/tls"
	"errors"
	"net"

	"golang.org/x/net/websocket"
)

// WebSocket implements Transport over an RFC 7395 XMPP-over-WebSocket
// connection. The XMPP stream is framed with <open/>/<close/> elements rather
// than <stream:stream> (handled by the negotiation layer when framing is
// enabled on the session).
type WebSocket struct {
	rwc  net.Conn
	tls  bool
	peer net.Addr
}

// NewWebSocket creates a new WebSocket transport over an established connection.
// A *websocket.Conn (which implements net.Conn) is accepted directly.
func NewWebSocket(conn net.Conn) *WebSocket {
	_, isTLS := conn.(*tls.Conn)
	return &WebSocket{
		rwc:  conn,
		tls:  isTLS,
		peer: conn.RemoteAddr(),
	}
}

// DialWebSocket connects to an XMPP WebSocket endpoint (ws:// or wss://) using
// the "xmpp" subprotocol (RFC 7395) and returns a Transport.
func DialWebSocket(url, origin string) (*WebSocket, error) {
	cfg, err := websocket.NewConfig(url, origin)
	if err != nil {
		return nil, err
	}
	cfg.Protocol = []string{"xmpp"}
	conn, err := websocket.DialConfig(cfg)
	if err != nil {
		return nil, err
	}
	conn.PayloadType = websocket.TextFrame
	return NewWebSocket(conn), nil
}

// Read reads data from the WebSocket connection.
func (ws *WebSocket) Read(p []byte) (int, error) {
	return ws.rwc.Read(p)
}

// Write writes data to the WebSocket connection.
func (ws *WebSocket) Write(p []byte) (int, error) {
	return ws.rwc.Write(p)
}

// Close closes the WebSocket connection.
func (ws *WebSocket) Close() error {
	return ws.rwc.Close()
}

// StartTLS returns an error because WebSocket connections use wss:// instead.
func (ws *WebSocket) StartTLS(_ *tls.Config) error {
	return errors.New("transport: WebSocket does not support STARTTLS; use wss://")
}

// ConnectionState returns the TLS state if the underlying connection is TLS.
func (ws *WebSocket) ConnectionState() (tls.ConnectionState, bool) {
	if tc, ok := ws.rwc.(*tls.Conn); ok {
		return tc.ConnectionState(), true
	}
	return tls.ConnectionState{}, false
}

// Peer returns the remote address.
func (ws *WebSocket) Peer() net.Addr {
	return ws.peer
}

// LocalAddress returns the local address.
func (ws *WebSocket) LocalAddress() net.Addr {
	return ws.rwc.LocalAddr()
}
