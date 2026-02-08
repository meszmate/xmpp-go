package xmpp

import (
	"bytes"
	"context"
	"errors"
	"sync"

	"github.com/meszmate/xmpp-go/dial"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/stanza"
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
	handler  Handler
}

// NewClient creates a new XMPP client.
func NewClient(addr jid.JID, password string, opts ...ClientOption) (*Client, error) {
	c := &Client{
		addr:     addr,
		password: password,
		dialer:   dial.NewDialer(),
	}

	for _, opt := range opts {
		opt.apply(&c.opts)
	}

	return c, nil
}

// Connect establishes a connection to the XMPP server.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	trans, err := c.dialer.Dial(ctx, c.addr.Domain())
	if err != nil {
		return err
	}

	sessionOpts := []SessionOption{
		WithLocalAddr(c.addr),
	}

	session, err := NewSession(ctx, trans, sessionOpts...)
	if err != nil {
		trans.Close()
		return err
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
		}
		if err := mgr.Initialize(ctx, params); err != nil {
			session.Close()
			c.session = nil
			return err
		}
		c.plugins = mgr
	}

	return nil
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
