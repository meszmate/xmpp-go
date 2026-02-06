package xmpp

import (
	"context"
	"crypto/tls"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	xmppxml "github.com/meszmate/xmpp-go/xml"
)

// StartTLS returns a StreamFeature for STARTTLS negotiation.
func StartTLS(config *tls.Config) StreamFeature {
	return StreamFeature{
		Name:       xml.Name{Space: ns.TLS, Local: "starttls"},
		Required:   true,
		Prohibited: StateSecure,
		List: func(ctx context.Context, e *xmppxml.Encoder) error {
			start := xml.StartElement{
				Name: xml.Name{Space: ns.TLS, Local: "starttls"},
			}
			if err := e.EncodeToken(start); err != nil {
				return err
			}
			req := xml.StartElement{Name: xml.Name{Local: "required"}}
			if err := e.EncodeToken(req); err != nil {
				return err
			}
			if err := e.EncodeToken(xml.EndElement{Name: req.Name}); err != nil {
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
			if err := session.Transport().StartTLS(config); err != nil {
				return 0, err
			}
			return StateSecure, nil
		},
	}
}
