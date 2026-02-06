// Package transport provides transport abstractions for XMPP connections.
package transport

import (
	"crypto/tls"
	"io"
	"net"
)

// Transport is the interface for XMPP connection transports.
type Transport interface {
	io.ReadWriteCloser

	// StartTLS upgrades the connection to TLS.
	StartTLS(config *tls.Config) error

	// ConnectionState returns the TLS connection state, if TLS is active.
	ConnectionState() (tls.ConnectionState, bool)

	// Peer returns the remote address.
	Peer() net.Addr

	// LocalAddress returns the local address.
	LocalAddress() net.Addr
}
