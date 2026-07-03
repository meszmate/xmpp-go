package xmpp

import (
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/storage"
)

// CertIdentityFunc maps a verified client certificate to the local username the
// server should authenticate it as. Returning false rejects the certificate.
type CertIdentityFunc func(cert *x509.Certificate) (username string, ok bool)

// serverOptions holds server configuration.
type serverOptions struct {
	addr           string
	tlsCert        string
	tlsKey         string
	tlsConfig      *tls.Config
	authFunc       AuthFunc
	sessionHandler SessionHandlerFunc
	storage        storage.Storage
	plugins        []plugin.Plugin
	pluginFactory  func() []plugin.Plugin
	allowAnonymous bool

	// SASL EXTERNAL (client-certificate) authentication.
	externalAuth CertIdentityFunc
	clientCAs    *x509.CertPool

	// Stream Management resumption window; zero means the default.
	resumeTimeout time.Duration

	// Server-to-server federation (XEP-0220 dialback).
	s2sSecret   string
	s2sResolver S2SResolver
}

// ServerOption configures a Server.
type ServerOption interface {
	apply(*serverOptions)
}

type serverOptionFunc func(*serverOptions)

func (f serverOptionFunc) apply(o *serverOptions) { f(o) }

// WithServerAddr sets the listen address.
func WithServerAddr(addr string) ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.addr = addr
	})
}

// WithServerTLS sets TLS certificate and key files.
func WithServerTLS(cert, key string) ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.tlsCert = cert
		o.tlsKey = key
	})
}

// WithServerAuth sets the authentication handler.
func WithServerAuth(f AuthFunc) ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.authFunc = f
	})
}

// WithServerSessionHandler sets the handler for new sessions.
func WithServerSessionHandler(f SessionHandlerFunc) ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.sessionHandler = f
	})
}

// WithServerStorage sets the pluggable storage backend.
func WithServerStorage(s storage.Storage) ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.storage = s
	})
}

// WithServerPlugins registers server-global plugins, initialized once when the
// server starts. Use these for shared services with server-wide state. They do
// not receive per-session inbound-stanza dispatch; for that use
// WithServerPluginFactory.
func WithServerPlugins(plugins ...plugin.Plugin) ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.plugins = append(o.plugins, plugins...)
	})
}

// WithServerPluginFactory registers a factory that produces a fresh set of
// plugins for each authenticated session. Each session's plugins are
// initialized with session-scoped send/handle hooks and receive inbound IQ
// dispatch (matched by payload namespace) after the built-in service handlers.
// Because a new instance is created per session, plugins may hold per-session
// state safely. The session's plugins are closed when the session ends.
func WithServerPluginFactory(factory func() []plugin.Plugin) ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.pluginFactory = factory
	})
}

// WithServerAnonymous enables the SASL ANONYMOUS mechanism, allowing clients to
// authenticate without credentials; each is assigned a random JID.
func WithServerAnonymous() ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.allowAnonymous = true
	})
}

// WithServerTLSConfig sets an explicit *tls.Config to use for STARTTLS instead
// of loading a certificate/key from files. It takes precedence over
// WithServerTLS. Useful for in-memory certificates and advanced TLS settings
// (client-certificate auth, custom cipher suites).
func WithServerTLSConfig(cfg *tls.Config) ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.tlsConfig = cfg
	})
}

// WithServerS2S enables server-to-server federation using XEP-0220 Server
// Dialback. secret is this server's private dialback secret (used to generate
// and verify dialback keys for its own domain), and resolver maps remote XMPP
// domains to dialable addresses (replacing DNS SRV lookup of
// `_xmpp-server._tcp`). Stanzas addressed to non-local domains are then routed
// over authenticated s2s streams, and inbound s2s streams are accepted and
// verified via dialback.
func WithServerS2S(secret string, resolver S2SResolver) ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.s2sSecret = secret
		o.s2sResolver = resolver
	})
}

// WithServerResumeTimeout sets how long a dropped but Stream-Management-
// resumable session is held for resumption before being torn down. The default
// is 120 seconds.
func WithServerResumeTimeout(d time.Duration) ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.resumeTimeout = d
	})
}

// WithServerClientCertAuth enables SASL EXTERNAL authentication via client
// certificates (RFC 6120 §6, mutual TLS). The server requests a client
// certificate during the TLS handshake, verifies it against caPool, and — when
// the client selects EXTERNAL — maps the presented certificate to a local
// username via identity. TLS must be configured (WithServerTLS or
// WithServerTLSConfig) for this to take effect.
func WithServerClientCertAuth(caPool *x509.CertPool, identity CertIdentityFunc) ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.clientCAs = caPool
		o.externalAuth = identity
	})
}
