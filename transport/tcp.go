package transport

import (
	"crypto/tls"
	"net"
	"sync"
)

// TCP implements Transport over a TCP connection.
type TCP struct {
	mu   sync.Mutex
	conn net.Conn
	tls  bool
}

// NewTCP creates a new TCP transport from an existing connection.
func NewTCP(conn net.Conn) *TCP {
	_, isTLS := conn.(*tls.Conn)
	return &TCP{conn: conn, tls: isTLS}
}

// Read reads data from the connection.
func (t *TCP) Read(p []byte) (int, error) {
	return t.conn.Read(p)
}

// Write writes data to the connection.
func (t *TCP) Write(p []byte) (int, error) {
	return t.conn.Write(p)
}

// Close closes the connection.
func (t *TCP) Close() error {
	return t.conn.Close()
}

// StartTLS upgrades the connection to TLS.
func (t *TCP) StartTLS(config *tls.Config) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	tlsConn := tls.Client(t.conn, config)
	if err := tlsConn.Handshake(); err != nil {
		return err
	}
	t.conn = tlsConn
	t.tls = true
	return nil
}

// ConnectionState returns the TLS connection state.
func (t *TCP) ConnectionState() (tls.ConnectionState, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if tlsConn, ok := t.conn.(*tls.Conn); ok {
		return tlsConn.ConnectionState(), true
	}
	return tls.ConnectionState{}, false
}

// Peer returns the remote address.
func (t *TCP) Peer() net.Addr {
	return t.conn.RemoteAddr()
}

// LocalAddress returns the local address.
func (t *TCP) LocalAddress() net.Addr {
	return t.conn.LocalAddr()
}

// Conn returns the underlying net.Conn.
func (t *TCP) Conn() net.Conn {
	return t.conn
}
