package xmpp

import (
	"crypto/tls"

	"github.com/meszmate/xmpp-go/dial"
)

type clientOptions struct {
	tlsConfig *tls.Config
	dialer    *dial.Dialer
	handler   Handler
	directTLS bool
	noTLS     bool
}

// ClientOption configures a Client.
type ClientOption interface {
	apply(*clientOptions)
}

type clientOptionFunc func(*clientOptions)

func (f clientOptionFunc) apply(o *clientOptions) { f(o) }

// WithClientTLS sets the TLS configuration for the client.
func WithClientTLS(config *tls.Config) ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.tlsConfig = config
	})
}

// WithClientDialer sets a custom dialer.
func WithClientDialer(d *dial.Dialer) ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.dialer = d
	})
}

// WithHandler sets the stanza handler for the client.
func WithHandler(h Handler) ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.handler = h
	})
}

// WithDirectTLS enables Direct TLS (XEP-0368).
func WithDirectTLS() ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.directTLS = true
	})
}

// WithNoTLS disables TLS (for testing only).
func WithNoTLS() ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.noTLS = true
	})
}
