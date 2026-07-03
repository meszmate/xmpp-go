package xmpp

import (
	"crypto/tls"

	"github.com/meszmate/xmpp-go/dial"
	"github.com/meszmate/xmpp-go/plugin"
)

type clientOptions struct {
	tlsConfig      *tls.Config
	dialer         *dial.Dialer
	handler        Handler
	directTLS      bool
	noTLS          bool
	resource       string
	lang           string
	connectAddr    string
	websocketURL   string
	boshURL        string
	saslMechanisms []string
	insecureSASL   bool
	noSM           bool
	noSMResume     bool
	plugins        []plugin.Plugin
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

// WithPlugins registers plugins to be initialized on connect.
func WithPlugins(plugins ...plugin.Plugin) ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.plugins = append(o.plugins, plugins...)
	})
}

// WithResource requests a specific resource for the bound JID. If empty (the
// default) the server assigns one.
func WithResource(resource string) ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.resource = resource
	})
}

// WithLang sets the default xml:lang advertised in the client stream header.
func WithLang(lang string) ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.lang = lang
	})
}

// WithConnectAddr pins the TCP address (host:port) the client dials, bypassing
// DNS SRV resolution of the JID domain. Useful for testing and for connecting
// to a server that is not discoverable via SRV.
func WithConnectAddr(addr string) ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.connectAddr = addr
	})
}

// WithWebSocket connects over XMPP-over-WebSocket (RFC 7395) to the given
// ws:// or wss:// endpoint instead of a TCP connection.
func WithWebSocket(url string) ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.websocketURL = url
	})
}

// WithBOSH connects over XMPP-over-BOSH (XEP-0124/0206) to the given HTTP(S)
// connection-manager endpoint (typically http(s)://host/http-bind) instead of a
// TCP connection.
func WithBOSH(url string) ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.boshURL = url
	})
}

// WithoutStreamManagement disables XEP-0198 Stream Management even when the
// server advertises it.
func WithoutStreamManagement() ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.noSM = true
	})
}

// WithoutStreamResumption keeps XEP-0198 Stream Management enabled (stanza
// acknowledgement) but does not request resumption support, so the session
// cannot be revived after a dropped connection.
func WithoutStreamResumption() ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.noSMResume = true
	})
}

// WithInsecureSASL permits password-bearing SASL mechanisms (PLAIN) over an
// unencrypted connection. This is unsafe and intended only for testing against
// a local server; by default the client refuses to send a cleartext password
// without TLS.
func WithInsecureSASL() ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.insecureSASL = true
	})
}

// WithSASLMechanisms overrides the client's SASL mechanism preference order.
// Only mechanisms in this list are attempted, most-preferred first. Recognized
// names: SCRAM-SHA-512, SCRAM-SHA-256, SCRAM-SHA-1, PLAIN. When unset the client
// prefers SCRAM-SHA-512 > SCRAM-SHA-256 > SCRAM-SHA-1 > PLAIN.
func WithSASLMechanisms(mechanisms ...string) ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.saslMechanisms = mechanisms
	})
}
