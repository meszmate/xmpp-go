package transport

import (
	"crypto/tls"
	"net"
	"sync"
	"time"
)

// tlsRole records whether StartTLS should perform the TLS client or server
// handshake. It is set explicitly by the negotiation layer; roleUnset falls back
// to a heuristic for backward compatibility.
type tlsRole int

const (
	roleUnset tlsRole = iota
	roleClient
	roleServer
)

// TCP implements Transport over a TCP connection.
type TCP struct {
	mu   sync.Mutex
	conn net.Conn
	tls  bool
	role tlsRole
}

// NewTCP creates a new TCP transport from an existing connection.
func NewTCP(conn net.Conn) *TCP {
	_, isTLS := conn.(*tls.Conn)
	return &TCP{conn: conn, tls: isTLS}
}

// SetTLSRole records whether a subsequent StartTLS should act as the TLS server
// (server=true) or client (server=false). This must be set for client-
// certificate (mutual TLS) authentication, where presenting a certificate would
// otherwise be misread as "act as a TLS server".
func (t *TCP) SetTLSRole(server bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if server {
		t.role = roleServer
	} else {
		t.role = roleClient
	}
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

	if config == nil {
		config = &tls.Config{}
	}

	var asServer bool
	switch t.role {
	case roleServer:
		asServer = true
	case roleClient:
		asServer = false
	default:
		// Backward-compatible heuristic when the role was not set explicitly: a
		// config carrying server certificate material implies the server side.
		asServer = len(config.Certificates) > 0 || config.GetCertificate != nil || config.GetConfigForClient != nil
	}

	var tlsConn *tls.Conn
	if asServer {
		tlsConn = tls.Server(t.conn, config)
	} else {
		tlsConn = tls.Client(t.conn, config)
	}

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

// SetDeadline sets the read and write deadline on the underlying connection.
// A zero time clears the deadline. Used to enforce a context deadline during
// stream negotiation.
func (t *TCP) SetDeadline(deadline time.Time) error {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()
	return conn.SetDeadline(deadline)
}
