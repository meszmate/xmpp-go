package xmpp

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/transport"
)

// Server is a high-level XMPP server.
type Server struct {
	mu                sync.Mutex
	domain            string
	listener          net.Listener
	sessions          map[string]*Session
	sessionPlugins    map[*Session]*plugin.Manager
	detached          map[string]*detachedEntry
	boshConns         map[string]*boshConn
	router            *localRouter
	tlsConfig         *tls.Config
	tlsServerEndpoint []byte // RFC 5929 tls-server-end-point CB for SCRAM-*-PLUS
	plugins           *plugin.Manager
	opts              serverOptions
	closed            chan struct{}

	s2sMu  sync.Mutex
	s2sOut map[string]*s2sOutbound
}

// NewServer creates a new XMPP server.
func NewServer(domain string, opts ...ServerOption) (*Server, error) {
	if domain == "" {
		return nil, errors.New("xmpp: server domain must not be empty")
	}
	s := &Server{
		domain:         domain,
		sessions:       make(map[string]*Session),
		sessionPlugins: make(map[*Session]*plugin.Manager),
		detached:       make(map[string]*detachedEntry),
		router:         newLocalRouter(),
		closed:         make(chan struct{}),
	}

	for _, opt := range opts {
		opt.apply(&s.opts)
	}

	return s, nil
}

// ListenAndServe starts listening for XMPP connections.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if st := s.opts.storage; st != nil {
		if err := st.Init(ctx); err != nil {
			return err
		}
		// Auto-derive AuthFunc from UserStore if no explicit authFunc is set.
		if s.opts.authFunc == nil {
			if us := st.UserStore(); us != nil {
				s.opts.authFunc = func(username, password string) (bool, error) {
					return us.Authenticate(ctx, username, password)
				}
			}
		}
	}

	if len(s.opts.plugins) > 0 {
		mgr := plugin.NewManager()
		for _, p := range s.opts.plugins {
			if err := mgr.Register(p); err != nil {
				return err
			}
		}
		params := plugin.InitParams{
			State:     func() uint32 { return uint32(StateServer) },
			LocalJID:  func() string { return s.domain },
			RemoteJID: func() string { return "" },
			Storage:   s.opts.storage,
		}
		if err := mgr.Initialize(ctx, params); err != nil {
			return err
		}
		s.plugins = mgr
	}

	addr := s.opts.addr
	if addr == "" {
		addr = ":5222"
	}

	var listener net.Listener
	var err error

	if s.opts.sessionHandler != nil && s.opts.tlsCert != "" && s.opts.tlsKey != "" {
		// Custom session handler with an implicit-TLS listener (legacy behavior:
		// the handler owns all negotiation).
		cert, certErr := tls.LoadX509KeyPair(s.opts.tlsCert, s.opts.tlsKey)
		if certErr != nil {
			return certErr
		}
		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
		listener, err = tls.Listen("tcp", addr, tlsConfig)
	} else {
		// Library-managed negotiation: listen on plain TCP and, when a
		// certificate is configured, offer STARTTLS to upgrade in-stream.
		if tlsCfg, tlsErr := s.buildServerTLS(); tlsErr != nil {
			return tlsErr
		} else {
			s.tlsConfig = tlsCfg
			s.tlsServerEndpoint = serverEndpointFromTLS(tlsCfg)
		}
		listener, err = net.Listen("tcp", addr)
	}

	if err != nil {
		return err
	}

	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()

	return s.serve(ctx, listener)
}

// buildServerTLS assembles the STARTTLS server configuration from the explicit
// tls.Config (WithServerTLSConfig) or a certificate/key pair (WithServerTLS),
// layering in client-certificate verification when SASL EXTERNAL is enabled.
// It returns nil (no error) when no TLS is configured.
func (s *Server) buildServerTLS() (*tls.Config, error) {
	var cfg *tls.Config
	switch {
	case s.opts.tlsConfig != nil:
		cfg = s.opts.tlsConfig.Clone()
	case s.opts.tlsCert != "" && s.opts.tlsKey != "":
		cert, err := tls.LoadX509KeyPair(s.opts.tlsCert, s.opts.tlsKey)
		if err != nil {
			return nil, err
		}
		cfg = &tls.Config{Certificates: []tls.Certificate{cert}}
	default:
		return nil, nil
	}
	if cfg.MinVersion == 0 {
		cfg.MinVersion = tls.VersionTLS12
	}
	// Enable SASL EXTERNAL: request (but do not force) a client certificate and
	// verify any presented certificate against the configured CA pool.
	if s.opts.externalAuth != nil {
		if cfg.ClientCAs == nil {
			cfg.ClientCAs = s.opts.clientCAs
		}
		if cfg.ClientAuth == tls.NoClientCert {
			cfg.ClientAuth = tls.VerifyClientCertIfGiven
		}
	}
	return cfg, nil
}

func (s *Server) serve(ctx context.Context, listener net.Listener) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.closed:
			return nil
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.closed:
				return nil
			default:
				return err
			}
		}

		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	trans := transport.NewTCP(conn)

	session, err := NewSession(ctx, trans,
		WithState(StateServer),
		WithRemoteAddr(jid.JID{}),
	)
	if err != nil {
		conn.Close()
		return
	}

	// An implicit-TLS listener yields an already-secure transport.
	if _, secure := trans.ConnectionState(); secure {
		session.SetState(StateSecure)
	}

	s.mu.Lock()
	s.sessions[conn.RemoteAddr().String()] = session
	s.mu.Unlock()

	defer func() {
		session.Close()
		s.mu.Lock()
		delete(s.sessions, conn.RemoteAddr().String())
		s.mu.Unlock()
	}()

	if s.opts.sessionHandler != nil {
		s.opts.sessionHandler(ctx, session)
		return
	}

	// No custom handler: run the library's built-in negotiation and routing.
	s.negotiateAndServe(ctx, session)
}

// Close stops the server.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.closed:
		return nil
	default:
		close(s.closed)
	}

	var firstErr error
	if s.listener != nil {
		if err := s.listener.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	for _, session := range s.sessions {
		if err := session.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if s.plugins != nil {
		if err := s.plugins.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		s.plugins = nil
	}

	if s.opts.storage != nil {
		if err := s.opts.storage.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// Plugin returns a registered plugin by name.
func (s *Server) Plugin(name string) (plugin.Plugin, bool) {
	s.mu.Lock()
	mgr := s.plugins
	s.mu.Unlock()

	if mgr == nil {
		return nil, false
	}
	return mgr.Get(name)
}

// Domain returns the server domain.
func (s *Server) Domain() string {
	return s.domain
}

// Addr returns the address the server is listening on, or nil if it is not yet
// listening.
func (s *Server) Addr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// SessionCount returns the number of active sessions.
func (s *Server) SessionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

// AuthFunc is a function that validates credentials.
type AuthFunc func(username, password string) (bool, error)

// SessionHandlerFunc is called when a new session is established.
type SessionHandlerFunc func(ctx context.Context, session *Session)
