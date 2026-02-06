package xmpp

import (
	"context"
	"errors"
	"sync"

	"github.com/meszmate/xmpp-go/dial"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
)

// Client is a high-level XMPP client.
type Client struct {
	mu       sync.Mutex
	addr     jid.JID
	password string
	session  *Session
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

	if c.session != nil {
		return c.session.Close()
	}
	return nil
}

// JID returns the client's JID.
func (c *Client) JID() jid.JID {
	return c.addr
}
