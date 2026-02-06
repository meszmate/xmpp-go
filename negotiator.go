package xmpp

import (
	"context"
)

// Negotiator handles XMPP stream negotiation.
type Negotiator struct {
	features []StreamFeature
}

// NewNegotiator creates a new stream negotiator.
func NewNegotiator(features ...StreamFeature) *Negotiator {
	return &Negotiator{features: features}
}

// AddFeature adds a stream feature to the negotiator.
func (n *Negotiator) AddFeature(f StreamFeature) {
	n.features = append(n.features, f)
}

// Features returns the features available for the given session state.
func (n *Negotiator) Features(state SessionState) []StreamFeature {
	var available []StreamFeature
	for _, f := range n.features {
		if f.Necessary != 0 && (state&f.Necessary) != f.Necessary {
			continue
		}
		if f.Prohibited != 0 && (state&f.Prohibited) != 0 {
			continue
		}
		available = append(available, f)
	}
	return available
}

// Negotiate performs stream feature negotiation on a session.
func (n *Negotiator) Negotiate(ctx context.Context, session *Session) error {
	// Negotiation is driven by the session type (client vs server).
	// The session calls individual feature's Negotiate methods.
	// This is a placeholder for the negotiation state machine.
	return nil
}
