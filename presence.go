package xmpp

import (
	"context"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/storage"
)

// dispatchPresence implements RFC 6121 presence handling: broadcast of
// available/unavailable presence to subscribed contacts, directed presence, and
// the subscription state machine (subscribe/subscribed/unsubscribe/unsubscribed)
// with roster updates and pushes.
func (s *Server) dispatchPresence(ctx context.Context, session *Session, pres *stanza.Presence) error {
	switch pres.Type {
	case stanza.PresenceAvailable, stanza.PresenceUnavailable:
		if pres.To.IsZero() {
			return s.broadcastPresence(ctx, session, pres)
		}
		pres.From = session.RemoteAddr()
		s.deliverPresenceTo(ctx, session, pres, pres.To)
		return nil
	case stanza.PresenceSubscribe, stanza.PresenceSubscribed,
		stanza.PresenceUnsubscribe, stanza.PresenceUnsubscribed:
		return s.handleSubscription(ctx, session, pres)
	case stanza.PresenceProbe:
		// Answer a probe with our directed presence is out of scope; ignore.
		return nil
	default:
		return nil
	}
}

// broadcastPresence delivers available/unavailable presence to every contact
// that is subscribed to the sender (roster subscription from/both) plus the
// sender's other resources.
func (s *Server) broadcastPresence(ctx context.Context, session *Session, pres *stanza.Presence) error {
	full := session.RemoteAddr()
	pres.From = full

	if rs := s.rosterStore(); rs != nil {
		items, err := rs.GetRosterItems(ctx, full.Bare().String())
		if err == nil {
			for _, it := range items {
				if it.Subscription != "from" && it.Subscription != "both" {
					continue
				}
				if cj, err := jid.Parse(it.ContactJID); err == nil {
					s.deliverPresenceTo(ctx, session, pres, cj.Bare())
				}
			}
		}
	}
	// Deliver to the sender's own other resources (self-presence).
	s.deliverPresenceTo(ctx, session, pres, full.Bare())
	return nil
}

// deliverPresenceTo routes a copy of pres to every session of `to`, stamping the
// recipient JID and skipping the originating session.
func (s *Server) deliverPresenceTo(ctx context.Context, source *Session, pres *stanza.Presence, to jid.JID) {
	for _, dst := range s.router.targets(to) {
		if dst == source {
			continue
		}
		cp := *pres
		cp.To = dst.RemoteAddr()
		_ = dst.Send(ctx, &cp)
	}
}

// handleSubscription processes a presence subscription stanza, stamping the bare
// sender JID, updating rosters, pushing roster changes, and routing the stanza.
func (s *Server) handleSubscription(ctx context.Context, session *Session, pres *stanza.Presence) error {
	from := session.RemoteAddr().Bare()
	to := pres.To.Bare()
	if to.IsZero() {
		return nil
	}
	pres.From = from
	rs := s.rosterStore()

	switch pres.Type {
	case stanza.PresenceSubscribe:
		// Sender wants to see `to`'s presence. Route the request; the target
		// decides. Record the outbound pending subscription.
		if rs != nil {
			s.updateRosterAsk(ctx, rs, from.String(), to.String(), "subscribe")
			s.pushRoster(ctx, from, to.String())
		}
		s.deliverPresenceTo(ctx, session, pres, to)

	case stanza.PresenceSubscribed:
		// Sender approves `to`'s request: `to` may now see the sender.
		if rs != nil {
			s.updateSubscription(ctx, rs, from.String(), to.String(), "from") // approver gains "from"
			s.updateSubscription(ctx, rs, to.String(), from.String(), "to")   // subscriber gains "to"
			s.clearRosterAsk(ctx, rs, to.String(), from.String())
			s.pushRoster(ctx, from, to.String())
			s.pushRoster(ctx, to, from.String())
		}
		s.deliverPresenceTo(ctx, session, pres, to)

	case stanza.PresenceUnsubscribe:
		if rs != nil {
			s.updateSubscription(ctx, rs, from.String(), to.String(), "-to")
			s.pushRoster(ctx, from, to.String())
		}
		s.deliverPresenceTo(ctx, session, pres, to)

	case stanza.PresenceUnsubscribed:
		if rs != nil {
			s.updateSubscription(ctx, rs, from.String(), to.String(), "-from")
			s.updateSubscription(ctx, rs, to.String(), from.String(), "-to")
			s.pushRoster(ctx, from, to.String())
			s.pushRoster(ctx, to, from.String())
		}
		s.deliverPresenceTo(ctx, session, pres, to)
	}
	return nil
}

// updateSubscription merges (or removes) a subscription direction on owner's
// roster item for contact. dir is "to"/"from" to add, or "-to"/"-from" to remove.
func (s *Server) updateSubscription(ctx context.Context, rs storage.RosterStore, owner, contact, dir string) {
	item := &storage.RosterItem{UserJID: owner, ContactJID: contact, Subscription: "none"}
	if existing, err := rs.GetRosterItem(ctx, owner, contact); err == nil && existing != nil {
		*item = *existing
	}
	switch dir {
	case "to", "from":
		item.Subscription = mergeSub(item.Subscription, dir)
	case "-to":
		item.Subscription = removeSub(item.Subscription, "to")
	case "-from":
		item.Subscription = removeSub(item.Subscription, "from")
	}
	_ = rs.UpsertRosterItem(ctx, item)
}

func (s *Server) updateRosterAsk(ctx context.Context, rs storage.RosterStore, owner, contact, ask string) {
	item := &storage.RosterItem{UserJID: owner, ContactJID: contact, Subscription: "none"}
	if existing, err := rs.GetRosterItem(ctx, owner, contact); err == nil && existing != nil {
		*item = *existing
	}
	item.Ask = ask
	_ = rs.UpsertRosterItem(ctx, item)
}

func (s *Server) clearRosterAsk(ctx context.Context, rs storage.RosterStore, owner, contact string) {
	if existing, err := rs.GetRosterItem(ctx, owner, contact); err == nil && existing != nil {
		existing.Ask = ""
		_ = rs.UpsertRosterItem(ctx, existing)
	}
}

// pushRoster sends an RFC 6121 roster push for a single contact to all of the
// owner's connected resources.
func (s *Server) pushRoster(ctx context.Context, owner jid.JID, contact string) {
	rs := s.rosterStore()
	if rs == nil {
		return
	}
	it, err := rs.GetRosterItem(ctx, owner.Bare().String(), contact)
	if err != nil || it == nil {
		return
	}
	q := rosterQuery{Items: []rosterItemXML{{
		JID:          it.ContactJID,
		Name:         it.Name,
		Subscription: it.Subscription,
		Ask:          it.Ask,
		Groups:       it.Groups,
	}}}
	for _, dst := range s.router.targets(owner.Bare()) {
		push := stanza.IQ{Header: stanza.Header{ID: randomID(), Type: stanza.IQSet, To: dst.RemoteAddr()}}
		_ = dst.SendElement(ctx, &stanza.IQPayload{IQ: push, Payload: &q})
	}
}

// mergeSub adds a subscription direction ("to" or "from") to a current state.
func mergeSub(cur, add string) string {
	switch cur {
	case "both":
		return "both"
	case "to":
		if add == "from" {
			return "both"
		}
		return "to"
	case "from":
		if add == "to" {
			return "both"
		}
		return "from"
	default: // none / empty
		return add
	}
}

// removeSub removes a subscription direction ("to" or "from") from a state.
func removeSub(cur, part string) string {
	switch cur {
	case "both":
		if part == "to" {
			return "from"
		}
		return "to"
	case part:
		return "none"
	default:
		return cur
	}
}
