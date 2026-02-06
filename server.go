package xmpp

import (
	"context"
	"crypto/tls"
	"net"
	"sync"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/transport"
)

// Server is a high-level XMPP server.
type Server struct {
	mu       sync.Mutex
	domain   string
	listener net.Listener
	sessions map[string]*Session
	opts     serverOptions
	closed   chan struct{}
}

// NewServer creates a new XMPP server.
func NewServer(domain string, opts ...ServerOption) (*Server, error) {
	s := &Server{
		domain:   domain,
		sessions: make(map[string]*Session),
		closed:   make(chan struct{}),
	}

	for _, opt := range opts {
		opt.apply(&s.opts)
	}

	return s, nil
}

// ListenAndServe starts listening for XMPP connections.
func (s *Server) ListenAndServe(ctx context.Context) error {
	addr := s.opts.addr
	if addr == "" {
		addr = ":5222"
	}

	var listener net.Listener
	var err error

	if s.opts.tlsCert != "" && s.opts.tlsKey != "" {
		cert, certErr := tls.LoadX509KeyPair(s.opts.tlsCert, s.opts.tlsKey)
		if certErr != nil {
			return certErr
		}
		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
		listener, err = tls.Listen("tcp", addr, tlsConfig)
	} else {
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
	}
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

	return firstErr
}

// Domain returns the server domain.
func (s *Server) Domain() string {
	return s.domain
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
