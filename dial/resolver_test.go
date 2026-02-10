package dial

import (
	"context"
	"fmt"
	"net"
	"testing"
)

func mockLookupSRV(records []*net.SRV, err error) func(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
	return func(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
		return "", records, err
	}
}

func TestResolveClient(t *testing.T) {
	t.Parallel()
	r := NewResolver()
	var capturedService string
	r.lookupSRV = func(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
		capturedService = service
		return "", []*net.SRV{
			{Target: "xmpp.example.com.", Port: 5222, Priority: 10, Weight: 50},
		}, nil
	}

	records, err := r.ResolveClient(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("ResolveClient: %v", err)
	}
	if capturedService != "xmpp-client" {
		t.Errorf("service = %q, want %q", capturedService, "xmpp-client")
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}
	if records[0].Port != 5222 {
		t.Errorf("Port = %d, want 5222", records[0].Port)
	}
}

func TestResolveServer(t *testing.T) {
	t.Parallel()
	r := NewResolver()
	var capturedService string
	r.lookupSRV = func(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
		capturedService = service
		return "", []*net.SRV{
			{Target: "xmpp.example.com.", Port: 5269, Priority: 10, Weight: 50},
		}, nil
	}

	_, err := r.ResolveServer(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("ResolveServer: %v", err)
	}
	if capturedService != "xmpp-server" {
		t.Errorf("service = %q, want %q", capturedService, "xmpp-server")
	}
}

func TestResolveClientTLS(t *testing.T) {
	t.Parallel()
	r := NewResolver()
	var capturedService string
	r.lookupSRV = func(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
		capturedService = service
		return "", []*net.SRV{
			{Target: "xmpp.example.com.", Port: 5223, Priority: 10, Weight: 50},
		}, nil
	}

	_, err := r.ResolveClientTLS(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("ResolveClientTLS: %v", err)
	}
	if capturedService != "xmpps-client" {
		t.Errorf("service = %q, want %q", capturedService, "xmpps-client")
	}
}

func TestResolveSortOrder(t *testing.T) {
	t.Parallel()
	r := NewResolver()
	r.lookupSRV = mockLookupSRV([]*net.SRV{
		{Target: "low-weight.example.com.", Port: 5222, Priority: 10, Weight: 10},
		{Target: "high-pri.example.com.", Port: 5222, Priority: 5, Weight: 50},
		{Target: "high-weight.example.com.", Port: 5222, Priority: 10, Weight: 90},
	}, nil)

	records, err := r.ResolveClient(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("ResolveClient: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("len(records) = %d, want 3", len(records))
	}
	// Priority 5 should come first
	if records[0].Priority != 5 {
		t.Errorf("records[0].Priority = %d, want 5", records[0].Priority)
	}
	// Among priority 10, higher weight should come first
	if records[1].Weight != 90 {
		t.Errorf("records[1].Weight = %d, want 90", records[1].Weight)
	}
	if records[2].Weight != 10 {
		t.Errorf("records[2].Weight = %d, want 10", records[2].Weight)
	}
}

func TestResolveFiltersDotTarget(t *testing.T) {
	t.Parallel()
	r := NewResolver()
	r.lookupSRV = mockLookupSRV([]*net.SRV{
		{Target: ".", Port: 5222, Priority: 10, Weight: 50},
		{Target: "xmpp.example.com.", Port: 5222, Priority: 10, Weight: 50},
	}, nil)

	records, err := r.ResolveClient(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("ResolveClient: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1 (dot target should be filtered)", len(records))
	}
	if records[0].Target != "xmpp.example.com." {
		t.Errorf("Target = %q", records[0].Target)
	}
}

func TestResolveLookupError(t *testing.T) {
	t.Parallel()
	r := NewResolver()
	r.lookupSRV = mockLookupSRV(nil, fmt.Errorf("dns failure"))

	_, err := r.ResolveClient(context.Background(), "example.com")
	if err == nil {
		t.Error("expected error from failed lookup")
	}
}
