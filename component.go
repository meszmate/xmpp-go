package xmpp

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/stream"
	"github.com/meszmate/xmpp-go/transport"
)

// Component implements the Jabber Component Protocol (XEP-0114).
type Component struct {
	mu      sync.Mutex
	domain  string
	secret  string
	session *Session
	addr    string
}

// NewComponent creates a new XMPP component.
func NewComponent(domain, secret string, opts ...ComponentOption) (*Component, error) {
	c := &Component{
		domain: domain,
		secret: secret,
		addr:   "localhost:5275",
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	return c, nil
}

// ComponentOption configures a Component.
type ComponentOption interface {
	apply(*Component)
}

type componentOptionFunc func(*Component)

func (f componentOptionFunc) apply(c *Component) { f(c) }

// WithComponentAddr sets the server address to connect to.
func WithComponentAddr(addr string) ComponentOption {
	return componentOptionFunc(func(c *Component) {
		c.addr = addr
	})
}

// Connect establishes a connection using the component protocol.
func (c *Component) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return fmt.Errorf("component: dial: %w", err)
	}

	trans := transport.NewTCP(conn)
	domainJID, err := jid.New("", c.domain, "")
	if err != nil {
		conn.Close()
		return err
	}

	session, err := NewSession(ctx, trans,
		WithLocalAddr(domainJID),
	)
	if err != nil {
		conn.Close()
		return err
	}

	// Send stream header
	header := stream.Open(stream.Header{
		To: domainJID,
		NS: "jabber:component:accept",
	})
	if _, err := session.Writer().WriteRaw(header); err != nil {
		session.Close()
		return err
	}

	c.session = session
	return nil
}

// Handshake generates the component handshake hash.
func (c *Component) Handshake(streamID string) string {
	h := sha1.New()
	h.Write([]byte(streamID + c.secret))
	return hex.EncodeToString(h.Sum(nil))
}

// Send sends a stanza via the component.
func (c *Component) Send(ctx context.Context, st stanza.Stanza) error {
	c.mu.Lock()
	s := c.session
	c.mu.Unlock()

	if s == nil {
		return errors.New("component: not connected")
	}
	return s.Send(ctx, st)
}

// Session returns the underlying session.
func (c *Component) Session() *Session {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.session
}

// Close closes the component connection.
func (c *Component) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != nil {
		return c.session.Close()
	}
	return nil
}

// Domain returns the component domain.
func (c *Component) Domain() string {
	return c.domain
}
