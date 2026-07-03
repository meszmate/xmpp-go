package xmpp

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/meszmate/xmpp-go/dial"
	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/transport"
)

// Client is a high-level XMPP client.
type Client struct {
	mu       sync.Mutex
	addr     jid.JID
	password string
	session  *Session
	plugins  *plugin.Manager
	dialer   *dial.Dialer
	opts     clientOptions
	serveErr chan error

	pendingMu sync.Mutex
	pending   map[string]chan *stanza.IQ
}

// NewClient creates a new XMPP client.
func NewClient(addr jid.JID, password string, opts ...ClientOption) (*Client, error) {
	c := &Client{
		addr:     addr,
		password: password,
		dialer:   dial.NewDialer(),
		pending:  make(map[string]chan *stanza.IQ),
	}

	for _, opt := range opts {
		opt.apply(&c.opts)
	}

	if addr.IsZero() {
		return nil, errors.New("xmpp: client JID must include a domain")
	}
	// A localpart is required unless the client authenticates anonymously.
	if addr.Local() == "" && !hasMechanism(c.opts.saslMechanisms, "ANONYMOUS") {
		return nil, errors.New("xmpp: client JID must include a localpart")
	}

	if c.opts.dialer != nil {
		c.dialer = c.opts.dialer
	}
	if c.opts.directTLS {
		c.dialer.DirectTLS = true
	}

	return c, nil
}

// Connect dials the server, performs full stream negotiation (STARTTLS, SASL
// authentication, and resource binding), initializes plugins, and starts the
// background receive loop.
//
// It returns only once the session has reached StateReady. Authentication
// failures are surfaced as *AuthError; a connection that cannot be established
// returns the underlying dial error.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != nil {
		return errors.New("xmpp: already connected")
	}

	trans, err := c.dialTransport(ctx)
	if err != nil {
		return err
	}
	session, err := c.newClientSession(ctx, trans)
	if err != nil {
		return err
	}
	return c.establish(ctx, trans, session, false)
}

// dialTransport establishes the underlying byte-stream transport for the client
// according to its configuration (WebSocket, a pinned TCP address, or SRV/dial).
func (c *Client) dialTransport(ctx context.Context) (transport.Transport, error) {
	switch {
	case c.opts.websocketURL != "":
		ws, err := transport.DialWebSocket(c.opts.websocketURL, "http://"+c.addr.Domain())
		if err != nil {
			return nil, err
		}
		return ws, nil
	case c.opts.boshURL != "":
		return transport.DialBOSHClient(c.opts.boshURL, c.addr.Domain())
	case c.opts.connectAddr != "":
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", c.opts.connectAddr)
		if err != nil {
			return nil, err
		}
		return transport.NewTCP(conn), nil
	default:
		return c.dialer.Dial(ctx, c.addr.Domain())
	}
}

// newClientSession wraps a transport in a session and records transport-derived
// state (WebSocket framing, existing TLS security).
func (c *Client) newClientSession(ctx context.Context, trans transport.Transport) (*Session, error) {
	session, err := NewSession(ctx, trans, WithLocalAddr(c.addr))
	if err != nil {
		trans.Close()
		return nil, err
	}
	if c.opts.websocketURL != "" {
		session.SetFraming(true)
		if strings.HasPrefix(c.opts.websocketURL, "wss") {
			session.SetState(StateSecure)
		}
	}
	if _, secure := trans.ConnectionState(); secure {
		session.SetState(StateSecure)
	}
	return session, nil
}

// establish runs negotiation (optionally resuming), initializes plugins, and
// starts the background receive loop. On success c.session is the live session.
func (c *Client) establish(ctx context.Context, trans transport.Transport, session *Session, resume bool) error {
	// Enforce the context deadline for the duration of negotiation so a stalled
	// server cannot hang forever. Cleared once negotiation completes.
	type deadliner interface{ SetDeadline(time.Time) error }
	if d, ok := trans.(deadliner); ok {
		if dl, hasDL := ctx.Deadline(); hasDL {
			_ = d.SetDeadline(dl)
		}
	}

	if err := c.negotiate(ctx, session, resume); err != nil {
		session.Close()
		return err
	}

	if d, ok := trans.(deadliner); ok {
		_ = d.SetDeadline(time.Time{})
	}
	c.session = session

	if len(c.opts.plugins) > 0 {
		mgr := plugin.NewManager()
		for _, p := range c.opts.plugins {
			if err := mgr.Register(p); err != nil {
				session.Close()
				c.session = nil
				return err
			}
		}
		params := plugin.InitParams{
			SendRaw: func(ctx context.Context, data []byte) error {
				return session.SendRaw(ctx, bytes.NewReader(data))
			},
			SendElement: session.SendElement,
			State:       func() uint32 { return uint32(session.State()) },
			LocalJID:    func() string { return session.LocalAddr().String() },
			RemoteJID:   func() string { return session.RemoteAddr().String() },
			Handle: func(name xml.Name, stanzaType string, h plugin.IncomingHandler) {
				session.Mux().Handle(name, stanzaType, HandlerFunc(func(ctx context.Context, _ *Session, st stanza.Stanza) error {
					return h(ctx, st)
				}))
			},
			Request: c.SendIQ,
		}
		if err := mgr.Initialize(ctx, params); err != nil {
			session.Close()
			c.session = nil
			return err
		}
		c.plugins = mgr
	}

	// A user-provided handler becomes the mux fallback so that plugin routes
	// registered on the session mux still fire; unmatched stanzas fall through
	// to the user handler.
	if c.opts.handler != nil {
		session.Mux().SetFallback(c.opts.handler)
	}

	// Start the background receive loop. Incoming stanzas are dispatched to the
	// session mux (plugin routes + user fallback). When the loop ends (EOF,
	// error, or Close), the session is closed so subsequent Send calls fail
	// rather than silently dropping into a dead socket.
	c.serveErr = make(chan error, 1)
	go func(s *Session, h Handler, done chan error) {
		err := s.Serve(h)
		_ = s.Close()
		done <- err
	}(session, c.receiveHandler(), c.serveErr)

	return nil
}

// Resume revives a dropped session using XEP-0198 §5 stream resumption: it dials
// a fresh connection, re-authenticates, and — if the previous session negotiated
// resumption — restores its bound resource and replays any unacknowledged
// stanzas rather than binding a new resource. If the server declines resumption,
// it transparently falls back to a fresh bind (a new resource).
//
// Resume is used after the receive loop reports the connection dropped (see
// Done). It requires that the prior session had resumption enabled.
func (c *Client) Resume(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	prev := c.session
	if prev == nil {
		return errors.New("xmpp: no session to resume")
	}
	if prev.SMPrevID() == "" {
		return errors.New("xmpp: session did not negotiate stream resumption")
	}

	trans, err := c.dialTransport(ctx)
	if err != nil {
		return err
	}
	session, err := c.newClientSession(ctx, trans)
	if err != nil {
		return err
	}
	// Carry the bound JID and Stream Management state (counters, unacked queue,
	// resumption id) onto the new stream so <resume/> and replay can proceed.
	session.SetLocalAddr(prev.LocalAddr())
	session.adoptSMState(prev)

	// Tear down the old plugin manager and session; a resumed stream gets fresh
	// client-side plugins via establish().
	if c.plugins != nil {
		_ = c.plugins.Close()
		c.plugins = nil
	}
	_ = prev.Close()
	c.session = nil

	return c.establish(ctx, trans, session, true)
}

// receiveHandler returns the effective inbound handler: it correlates
// result/error IQs to pending SendIQ waiters, auto-answers XEP-0199 pings (so
// the peer does not consider the client dead), and otherwise dispatches through
// the session mux (plugin routes plus any user-handler fallback).
func (c *Client) receiveHandler() Handler {
	return HandlerFunc(func(ctx context.Context, s *Session, st stanza.Stanza) error {
		if iq, ok := st.(*stanza.IQ); ok {
			// Deliver result/error IQs to any pending SendIQ waiter.
			if iq.Type == stanza.IQResult || iq.Type == stanza.IQError {
				c.pendingMu.Lock()
				ch, found := c.pending[iq.ID]
				if found {
					delete(c.pending, iq.ID)
				}
				c.pendingMu.Unlock()
				if found {
					ch <- iq
					return nil
				}
			}
			// Auto-answer XEP-0199 pings so the peer does not consider us dead.
			if iq.Type == stanza.IQGet && iqPayloadName(iq).Space == ns.Ping {
				res := stanza.IQ{Header: stanza.Header{ID: iq.ID, Type: stanza.IQResult, To: iq.From, From: s.LocalAddr()}}
				return s.SendElement(ctx, &stanza.IQPayload{IQ: res})
			}
		}
		return s.Mux().HandleStanza(ctx, s, st)
	})
}

// SendIQ sends an IQ request and waits for the correlated result/error IQ, or
// until ctx is done. The request ID is generated if empty.
func (c *Client) SendIQ(ctx context.Context, iq *stanza.IQ) (*stanza.IQ, error) {
	if iq.ID == "" {
		iq.ID = stanza.GenerateID()
	}
	ch := make(chan *stanza.IQ, 1)

	c.pendingMu.Lock()
	c.pending[iq.ID] = ch
	c.pendingMu.Unlock()
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, iq.ID)
		c.pendingMu.Unlock()
	}()

	if err := c.Send(ctx, iq); err != nil {
		return nil, err
	}
	select {
	case res := <-ch:
		return res, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Done returns a channel that receives the receive-loop error when the session
// terminates (nil on a clean stream close). It is valid after Connect returns.
func (c *Client) Done() <-chan error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.serveErr
}

// Send sends a stanza.
func (c *Client) Send(ctx context.Context, st stanza.Stanza) error {
	c.mu.Lock()
	s := c.session
	c.mu.Unlock()

	if s == nil {
		return errors.New("xmpp: not connected")
	}
	return s.Send(ctx, st)
}

// Session returns the underlying session.
func (c *Client) Session() *Session {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.session
}

// Close closes the client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var firstErr error
	if c.plugins != nil {
		if err := c.plugins.Close(); err != nil {
			firstErr = err
		}
		c.plugins = nil
	}
	if c.session != nil {
		if err := c.session.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		c.session = nil
	}
	return firstErr
}

// Plugin returns a registered plugin by name.
func (c *Client) Plugin(name string) (plugin.Plugin, bool) {
	c.mu.Lock()
	mgr := c.plugins
	c.mu.Unlock()

	if mgr == nil {
		return nil, false
	}
	return mgr.Get(name)
}

// JID returns the client's JID.
func (c *Client) JID() jid.JID {
	return c.addr
}
