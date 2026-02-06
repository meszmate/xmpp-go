package xmpp

import (
	"context"
	"encoding/xml"

	xmppxml "github.com/meszmate/xmpp-go/xml"
)

// StreamFeature represents a feature that can be negotiated during stream setup.
type StreamFeature struct {
	// Name is the XML name of the feature element.
	Name xml.Name

	// Required indicates this feature must be negotiated.
	Required bool

	// Necessary is the state required before this feature can be negotiated.
	Necessary SessionState

	// Prohibited is the state that prevents this feature from being negotiated.
	Prohibited SessionState

	// List writes the feature advertisement to the stream.
	List func(ctx context.Context, e *xmppxml.Encoder) error

	// Parse reads the feature data from the stream.
	Parse func(ctx context.Context, d *xmppxml.Decoder, start *xml.StartElement) (any, error)

	// Negotiate performs the feature negotiation.
	Negotiate func(ctx context.Context, session *Session, data any) (SessionState, error)
}
