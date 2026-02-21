package dial

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/meszmate/xmpp-go/transport"
)

// Dialer dials XMPP connections.
type Dialer struct {
	Resolver  *Resolver
	TLSConfig *tls.Config
	Timeout   time.Duration
	DirectTLS bool
}

// NewDialer creates a new Dialer with default settings.
func NewDialer() *Dialer {
	return &Dialer{
		Resolver: NewResolver(),
		Timeout:  30 * time.Second,
	}
}

// Dial connects to an XMPP server for the given domain.
func (d *Dialer) Dial(ctx context.Context, domain string) (*transport.TCP, error) {
	if d.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.Timeout)
		defer cancel()
	}

	var records []SRVRecord
	var err error

	if d.DirectTLS {
		records, err = d.Resolver.ResolveClientTLS(ctx, domain)
	} else {
		records, err = d.Resolver.ResolveClient(ctx, domain)
	}

	// Fall back to domain:5222 if SRV lookup fails
	if err != nil || len(records) == 0 {
		port := "5222"
		if d.DirectTLS {
			port = "5223"
		}
		records = []SRVRecord{{Target: domain, Port: parsePort(port)}}
	}

	// Try each record in order
	var lastErr error
	netDialer := &net.Dialer{Timeout: d.Timeout}
	for _, rec := range records {
		addr := net.JoinHostPort(rec.Target, fmt.Sprintf("%d", rec.Port))

		var conn net.Conn
		if d.DirectTLS {
			tlsCfg := d.tlsConfig(domain)
			tlsDialer := &tls.Dialer{
				NetDialer: netDialer,
				Config:    tlsCfg,
			}
			conn, lastErr = tlsDialer.DialContext(ctx, "tcp", addr)
		} else {
			conn, lastErr = netDialer.DialContext(ctx, "tcp", addr)
		}

		if lastErr == nil {
			return transport.NewTCP(conn), nil
		}
	}

	return nil, fmt.Errorf("dial: failed to connect to %s: %w", domain, lastErr)
}

// DialServer connects to an XMPP server for S2S communication.
func (d *Dialer) DialServer(ctx context.Context, domain string) (*transport.TCP, error) {
	if d.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.Timeout)
		defer cancel()
	}

	records, err := d.Resolver.ResolveServer(ctx, domain)
	if err != nil || len(records) == 0 {
		records = []SRVRecord{{Target: domain, Port: 5269}}
	}

	var lastErr error
	netDialer := &net.Dialer{Timeout: d.Timeout}
	for _, rec := range records {
		addr := net.JoinHostPort(rec.Target, fmt.Sprintf("%d", rec.Port))
		conn, dialErr := netDialer.DialContext(ctx, "tcp", addr)
		if dialErr == nil {
			return transport.NewTCP(conn), nil
		}
		lastErr = dialErr
	}

	return nil, fmt.Errorf("dial: failed to connect to %s: %w", domain, lastErr)
}

func (d *Dialer) tlsConfig(domain string) *tls.Config {
	if d.TLSConfig != nil {
		cfg := d.TLSConfig.Clone()
		if cfg.ServerName == "" {
			cfg.ServerName = domain
		}
		return cfg
	}
	return &tls.Config{ServerName: domain}
}

func parsePort(s string) uint16 {
	var port uint16
	fmt.Sscanf(s, "%d", &port)
	return port
}
