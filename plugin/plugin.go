// Package plugin defines the XMPP plugin interface and registry.
package plugin

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/storage"
)

// IncomingHandler processes an inbound stanza routed to a plugin.
type IncomingHandler func(ctx context.Context, st stanza.Stanza) error

// Plugin is the interface that all XMPP plugins must implement.
type Plugin interface {
	// Name returns the unique plugin name.
	Name() string

	// Version returns the plugin version.
	Version() string

	// Initialize is called when the plugin is activated on a session.
	Initialize(ctx context.Context, params InitParams) error

	// Close releases resources held by the plugin.
	Close() error

	// Dependencies returns the names of plugins this plugin depends on.
	Dependencies() []string
}

// InitParams provides parameters for plugin initialization.
// This avoids a circular import with the root xmpp package.
type InitParams struct {
	// SendRaw sends raw bytes on the session.
	SendRaw func(ctx context.Context, data []byte) error
	// SendElement encodes and sends an XML element.
	SendElement func(ctx context.Context, v any) error
	// State returns the current session state as a uint32.
	State func() uint32
	// LocalJID returns the local JID string.
	LocalJID func() string
	// RemoteJID returns the remote JID string.
	RemoteJID func() string
	// Get retrieves another plugin by name.
	Get func(name string) (Plugin, bool)
	// Storage provides access to the pluggable storage layer. May be nil.
	Storage storage.Storage
	// Handle registers an inbound stanza handler. For IQ stanzas the name is
	// matched against the payload (child) element; for message/presence it is
	// matched against the stanza element. An empty name or stanzaType is a
	// wildcard. May be nil when the host does not support inbound dispatch.
	Handle func(name xml.Name, stanzaType string, h IncomingHandler)
	// Request sends an IQ and waits for the correlated reply. May be nil when
	// the host does not support request/response (e.g. a server session).
	Request func(ctx context.Context, iq *stanza.IQ) (*stanza.IQ, error)
}
