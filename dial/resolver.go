// Package dial provides connection dialing with DNS SRV and host-meta resolution.
package dial

import (
	"context"
	"fmt"
	"net"
	"sort"
)

// SRVRecord represents a resolved SRV record.
type SRVRecord struct {
	Target   string
	Port     uint16
	Priority uint16
	Weight   uint16
}

// Resolver resolves XMPP server addresses via DNS SRV records.
type Resolver struct {
	lookupSRV func(ctx context.Context, service, proto, name string) (string, []*net.SRV, error)
}

// NewResolver creates a new Resolver.
func NewResolver() *Resolver {
	return &Resolver{
		lookupSRV: net.DefaultResolver.LookupSRV,
	}
}

// ResolveClient resolves client-to-server SRV records for a domain.
func (r *Resolver) ResolveClient(ctx context.Context, domain string) ([]SRVRecord, error) {
	return r.resolve(ctx, "xmpp-client", "tcp", domain)
}

// ResolveServer resolves server-to-server SRV records for a domain.
func (r *Resolver) ResolveServer(ctx context.Context, domain string) ([]SRVRecord, error) {
	return r.resolve(ctx, "xmpp-server", "tcp", domain)
}

// ResolveClientTLS resolves Direct TLS client SRV records (XEP-0368).
func (r *Resolver) ResolveClientTLS(ctx context.Context, domain string) ([]SRVRecord, error) {
	return r.resolve(ctx, "xmpps-client", "tcp", domain)
}

// ResolveServerTLS resolves Direct TLS server SRV records (XEP-0368).
func (r *Resolver) ResolveServerTLS(ctx context.Context, domain string) ([]SRVRecord, error) {
	return r.resolve(ctx, "xmpps-server", "tcp", domain)
}

func (r *Resolver) resolve(ctx context.Context, service, proto, name string) ([]SRVRecord, error) {
	_, addrs, err := r.lookupSRV(ctx, service, proto, name)
	if err != nil {
		return nil, fmt.Errorf("dial: SRV lookup for _%s._%s.%s: %w", service, proto, name, err)
	}

	records := make([]SRVRecord, 0, len(addrs))
	for _, addr := range addrs {
		// A target of "." means the service is not available
		if addr.Target == "." {
			continue
		}
		records = append(records, SRVRecord{
			Target:   addr.Target,
			Port:     addr.Port,
			Priority: addr.Priority,
			Weight:   addr.Weight,
		})
	}

	// Sort by priority (ascending), then weight (descending)
	sort.Slice(records, func(i, j int) bool {
		if records[i].Priority != records[j].Priority {
			return records[i].Priority < records[j].Priority
		}
		return records[i].Weight > records[j].Weight
	})

	return records, nil
}
