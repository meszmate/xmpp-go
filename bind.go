package xmpp

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	xmppxml "github.com/meszmate/xmpp-go/xml"
)

// BindFeature returns a StreamFeature for resource binding.
func BindFeature() StreamFeature {
	return StreamFeature{
		Name:       xml.Name{Space: ns.Bind, Local: "bind"},
		Required:   true,
		Necessary:  StateAuthenticated,
		Prohibited: StateBound,
		List: func(ctx context.Context, e *xmppxml.Encoder) error {
			start := xml.StartElement{
				Name: xml.Name{Space: ns.Bind, Local: "bind"},
			}
			if err := e.EncodeToken(start); err != nil {
				return err
			}
			return e.EncodeToken(xml.EndElement{Name: start.Name})
		},
		Parse: func(ctx context.Context, d *xmppxml.Decoder, start *xml.StartElement) (any, error) {
			if err := d.Skip(); err != nil {
				return nil, err
			}
			return nil, nil
		},
		Negotiate: func(ctx context.Context, session *Session, data any) (SessionState, error) {
			// Resource binding handled by the client/server layer
			return StateBound | StateReady, nil
		},
	}
}

// BindRequest represents a resource bind request.
type BindRequest struct {
	XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
	Resource string   `xml:"resource,omitempty"`
}

// BindResult represents a resource bind result.
type BindResult struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
	JID     string   `xml:"jid"`
}
