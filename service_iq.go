package xmpp

import (
	"bytes"
	"context"
	"encoding/xml"
	"runtime"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/storage"
)

// LibraryVersion is reported by the server in XEP-0092 Software Version replies.
const LibraryVersion = "0.1.0"

// iqPayloadName returns the XML name of an IQ's first child (payload) element.
func iqPayloadName(iq *stanza.IQ) xml.Name {
	dec := xml.NewDecoder(bytes.NewReader(iq.Query))
	for {
		tok, err := dec.Token()
		if err != nil {
			return xml.Name{}
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name
		}
	}
}

// isServiceDirected reports whether an IQ is addressed to the server itself or
// to the sender's own account (both handled by the server directly).
func (s *Server) isServiceDirected(session *Session, iq *stanza.IQ) bool {
	to := iq.To
	if to.IsZero() {
		return true
	}
	if to.Local() == "" && to.Domain() == s.domain {
		return true
	}
	if to.Bare().Equal(session.RemoteAddr().Bare()) {
		return true
	}
	return false
}

// serverServiceIQ answers IQs directed at the server or the sender's account:
// disco#info/#items, XEP-0199 ping, XEP-0092 version, and RFC 6121 roster.
func (s *Server) serverServiceIQ(ctx context.Context, session *Session, iq *stanza.IQ) error {
	name := iqPayloadName(iq)
	switch {
	case iq.Type == stanza.IQGet && name.Space == ns.Ping:
		return s.serviceResult(ctx, session, iq, nil) // pong
	case iq.Type == stanza.IQGet && name.Space == ns.DiscoInfo:
		return s.serviceResult(ctx, session, iq, discoInfoResult{
			Identity: []discoIdentity{{Category: "server", Type: "im", Name: "xmpp-go"}},
			Feature: []discoFeature{
				{Var: ns.DiscoInfo}, {Var: ns.DiscoItems}, {Var: ns.Ping},
				{Var: ns.Version}, {Var: ns.Roster},
			},
		})
	case iq.Type == stanza.IQGet && name.Space == ns.DiscoItems:
		return s.serviceResult(ctx, session, iq, discoItemsResult{})
	case iq.Type == stanza.IQGet && name.Space == ns.Version:
		return s.serviceResult(ctx, session, iq, versionResult{
			Name: "xmpp-go", Version: LibraryVersion, OS: runtime.GOOS,
		})
	case name.Space == ns.Roster:
		return s.serverRoster(ctx, session, iq)
	default:
		if iq.Type == stanza.IQGet || iq.Type == stanza.IQSet {
			return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorFeatureNotImplemented, "unsupported service iq")))
		}
		return nil
	}
}

// serviceResult sends a result IQ carrying the given payload (nil for an empty
// result such as a pong).
func (s *Server) serviceResult(ctx context.Context, session *Session, iq *stanza.IQ, payload any) error {
	from, _ := jid.New("", s.domain, "")
	res := stanza.IQ{Header: stanza.Header{
		ID:   iq.ID,
		Type: stanza.IQResult,
		From: from,
		To:   session.RemoteAddr(),
	}}
	return session.SendElement(ctx, &stanza.IQPayload{IQ: res, Payload: payload})
}

func (s *Server) serverRoster(ctx context.Context, session *Session, iq *stanza.IQ) error {
	rs := s.rosterStore()
	if rs == nil {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorServiceUnavailable, "roster storage unavailable")))
	}
	user := session.RemoteAddr().Bare().String()

	switch iq.Type {
	case stanza.IQGet:
		items, err := rs.GetRosterItems(ctx, user)
		if err != nil {
			return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorInternalServerError, "roster read failed")))
		}
		return s.serviceResult(ctx, session, iq, rosterQueryFromItems(items))

	case stanza.IQSet:
		var q rosterQuery
		if err := xml.Unmarshal(iq.Query, &q); err != nil || len(q.Items) == 0 {
			return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeModify, stanza.ErrorBadRequest, "invalid roster set")))
		}
		for _, it := range q.Items {
			if it.JID == "" {
				continue
			}
			if it.Subscription == "remove" {
				_ = rs.DeleteRosterItem(ctx, user, it.JID)
				continue
			}
			_ = rs.UpsertRosterItem(ctx, &storage.RosterItem{
				UserJID:      user,
				ContactJID:   it.JID,
				Name:         it.Name,
				Subscription: it.Subscription,
				Groups:       it.Groups,
			})
		}
		return s.serviceResult(ctx, session, iq, nil)
	}
	return nil
}

func (s *Server) rosterStore() storage.RosterStore {
	if s.opts.storage == nil {
		return nil
	}
	return s.opts.storage.RosterStore()
}

// --- service IQ payload types ---

type discoInfoResult struct {
	XMLName  xml.Name        `xml:"http://jabber.org/protocol/disco#info query"`
	Identity []discoIdentity `xml:"identity"`
	Feature  []discoFeature  `xml:"feature"`
}

type discoIdentity struct {
	Category string `xml:"category,attr"`
	Type     string `xml:"type,attr"`
	Name     string `xml:"name,attr,omitempty"`
}

type discoFeature struct {
	Var string `xml:"var,attr"`
}

type discoItemsResult struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/disco#items query"`
}

type versionResult struct {
	XMLName xml.Name `xml:"jabber:iq:version query"`
	Name    string   `xml:"name"`
	Version string   `xml:"version"`
	OS      string   `xml:"os,omitempty"`
}

type rosterQuery struct {
	XMLName xml.Name        `xml:"jabber:iq:roster query"`
	Items   []rosterItemXML `xml:"item"`
}

type rosterItemXML struct {
	JID          string   `xml:"jid,attr"`
	Name         string   `xml:"name,attr,omitempty"`
	Subscription string   `xml:"subscription,attr,omitempty"`
	Ask          string   `xml:"ask,attr,omitempty"`
	Groups       []string `xml:"group"`
}

func rosterQueryFromItems(items []*storage.RosterItem) rosterQuery {
	q := rosterQuery{}
	for _, it := range items {
		q.Items = append(q.Items, rosterItemXML{
			JID:          it.ContactJID,
			Name:         it.Name,
			Subscription: it.Subscription,
			Ask:          it.Ask,
			Groups:       it.Groups,
		})
	}
	return q
}
