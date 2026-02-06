// Package plugin defines the XMPP plugin interface and registry.
package plugin

import (
	"context"
)

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
}
