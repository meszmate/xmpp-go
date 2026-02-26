package main

import (
	"context"
	"encoding/xml"
	"io"
	"log"

	xmpp "github.com/meszmate/xmpp-go"
	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/storage"
	xmppxml "github.com/meszmate/xmpp-go/xml"
)

func serveSession(ctx context.Context, session *xmpp.Session, cfg Config, store storage.Storage) {
	mux := session.Mux()
	regHandler := newRegistrationHandler(cfg.Registration, store)
	mux.HandleFunc(xml.Name{Local: "iq"}, stanza.IQGet, regHandler.Handle)
	mux.HandleFunc(xml.Name{Local: "iq"}, stanza.IQSet, regHandler.Handle)

	if err := serveStream(ctx, session, mux, cfg); err != nil {
		log.Printf("session error: %v", err)
	}
}

func serveStream(ctx context.Context, session *xmpp.Session, handler xmpp.Handler, cfg Config) error {
	reader := session.Reader()
	writer := session.Writer()
	streamOpened := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		tok, err := reader.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		if start.Name.Space == ns.Stream && start.Name.Local == "stream" {
			if !streamOpened {
				if err := writeStreamStart(writer, cfg.Domain); err != nil {
					return err
				}
				if err := writeStreamFeatures(writer, cfg); err != nil {
					return err
				}
				streamOpened = true
			}
			continue
		}

		switch start.Name.Local {
		case "message":
			msg := &stanza.Message{}
			if err := reader.DecodeElement(msg, &start); err != nil {
				return err
			}
			if err := handler.HandleStanza(context.Background(), session, msg); err != nil {
				return err
			}
		case "presence":
			pres := &stanza.Presence{}
			if err := reader.DecodeElement(pres, &start); err != nil {
				return err
			}
			if err := handler.HandleStanza(context.Background(), session, pres); err != nil {
				return err
			}
		case "iq":
			iq := &stanza.IQ{}
			if err := reader.DecodeElement(iq, &start); err != nil {
				return err
			}
			if err := handler.HandleStanza(context.Background(), session, iq); err != nil {
				return err
			}
		default:
			if err := reader.Skip(); err != nil {
				return err
			}
		}
	}
}

func writeStreamStart(writer *xmppxml.StreamWriter, domain string) error {
	start := xml.StartElement{
		Name: xml.Name{Space: ns.Stream, Local: "stream"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: ns.Client},
			{Name: xml.Name{Local: "xmlns:stream"}, Value: ns.Stream},
			{Name: xml.Name{Local: "from"}, Value: domain},
			{Name: xml.Name{Local: "version"}, Value: "1.0"},
		},
	}
	return writer.EncodeToken(start)
}

func writeStreamFeatures(writer *xmppxml.StreamWriter, cfg Config) error {
	start := xml.StartElement{Name: xml.Name{Space: ns.Stream, Local: "features"}}
	if err := writer.EncodeToken(start); err != nil {
		return err
	}

	if cfg.Registration.Policy != registrationClosed {
		feature := xml.StartElement{Name: xml.Name{Space: ns.Register, Local: "register"}}
		if err := writer.EncodeToken(feature); err != nil {
			return err
		}
		if err := writer.EncodeToken(xml.EndElement{Name: feature.Name}); err != nil {
			return err
		}
	}

	return writer.EncodeToken(xml.EndElement{Name: start.Name})
}
