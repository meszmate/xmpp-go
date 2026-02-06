package xmpp

// serverOptions holds server configuration.
type serverOptions struct {
	addr           string
	tlsCert        string
	tlsKey         string
	authFunc       AuthFunc
	sessionHandler SessionHandlerFunc
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
