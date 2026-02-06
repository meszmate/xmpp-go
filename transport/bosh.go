package transport

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// BOSH implements Transport over BOSH (XEP-0124/0206).
type BOSH struct {
	mu       sync.Mutex
	url      string
	sid      string
	rid      int64
	client   *http.Client
	incoming *bytes.Buffer
	closed   bool
}

// NewBOSH creates a new BOSH transport.
func NewBOSH(url string) *BOSH {
	return &BOSH{
		url: url,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		incoming: new(bytes.Buffer),
	}
}

// Read reads data received from the BOSH connection.
func (b *BOSH) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return 0, io.EOF
	}
	return b.incoming.Read(p)
}

// Write sends data over the BOSH connection.
func (b *BOSH) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return 0, errors.New("transport: BOSH connection closed")
	}

	resp, err := b.client.Post(b.url, "text/xml; charset=utf-8", bytes.NewReader(p))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	b.incoming.Write(body)
	return len(p), nil
}

// Close closes the BOSH connection.
func (b *BOSH) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}

// StartTLS is not supported for BOSH (uses HTTPS instead).
func (b *BOSH) StartTLS(_ *tls.Config) error {
	return errors.New("transport: BOSH does not support STARTTLS; use HTTPS")
}

// ConnectionState returns empty state; BOSH uses HTTPS for security.
func (b *BOSH) ConnectionState() (tls.ConnectionState, bool) {
	return tls.ConnectionState{}, false
}

// Peer returns nil for BOSH as there's no direct peer connection.
func (b *BOSH) Peer() net.Addr {
	return nil
}

// LocalAddress returns nil for BOSH.
func (b *BOSH) LocalAddress() net.Addr {
	return nil
}

// SetSID sets the BOSH session ID.
func (b *BOSH) SetSID(sid string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sid = sid
}

// SID returns the BOSH session ID.
func (b *BOSH) SID() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sid
}
