package xmpp

import (
	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/storage"
)

// serverOptions holds server configuration.
type serverOptions struct {
	addr           string
	tlsCert        string
	tlsKey         string
	authFunc       AuthFunc
	sessionHandler SessionHandlerFunc
	storage        storage.Storage
	plugins        []plugin.Plugin
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

// WithServerPlugins registers plugins to be initialized on serve.
func WithServerPlugins(plugins ...plugin.Plugin) ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.plugins = append(o.plugins, plugins...)
	})
}
