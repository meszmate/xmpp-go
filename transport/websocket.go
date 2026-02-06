package transport

import (
	"crypto/tls"
	"errors"
	"net"
)

// WebSocket implements Transport over a WebSocket connection (RFC 7395).
// This is a structural implementation; actual WebSocket I/O requires
// a WebSocket library to be plugged in via the ReadWriteCloser.
type WebSocket struct {
	rwc  net.Conn
	tls  bool
	peer net.Addr
}

// NewWebSocket creates a new WebSocket transport.
func NewWebSocket(conn net.Conn) *WebSocket {
	_, isTLS := conn.(*tls.Conn)
	return &WebSocket{
		rwc:  conn,
		tls:  isTLS,
		peer: conn.RemoteAddr(),
	}
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
