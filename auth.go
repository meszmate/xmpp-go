package xmpp

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	xmppxml "github.com/meszmate/xmpp-go/xml"
)

// SASLFeature returns a StreamFeature for SASL authentication.
func SASLFeature(mechanisms []string) StreamFeature {
	return StreamFeature{
		Name:       xml.Name{Space: ns.SASL, Local: "mechanisms"},
		Required:   true,
		Necessary:  StateSecure,
		Prohibited: StateAuthenticated,
		List: func(ctx context.Context, e *xmppxml.Encoder) error {
			start := xml.StartElement{
				Name: xml.Name{Space: ns.SASL, Local: "mechanisms"},
			}
			if err := e.EncodeToken(start); err != nil {
				return err
			}
			for _, mech := range mechanisms {
				mechStart := xml.StartElement{Name: xml.Name{Space: ns.SASL, Local: "mechanism"}}
				if err := e.EncodeToken(mechStart); err != nil {
					return err
				}
				if err := e.EncodeToken(xml.CharData(mech)); err != nil {
					return err
				}
				if err := e.EncodeToken(xml.EndElement{Name: mechStart.Name}); err != nil {
					return err
				}
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
			// SASL negotiation handled by the client/server layer
			return StateAuthenticated, nil
		},
	}
}
