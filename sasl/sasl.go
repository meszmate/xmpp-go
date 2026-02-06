// Package sasl implements SASL authentication mechanisms for XMPP.
package sasl

import "errors"

var (
	ErrNoMechanism    = errors.New("sasl: no supported mechanism")
	ErrAuthFailed     = errors.New("sasl: authentication failed")
	ErrInvalidResponse = errors.New("sasl: invalid server response")
	ErrChannelBinding  = errors.New("sasl: channel binding not supported")
)

// Mechanism defines a SASL authentication mechanism.
type Mechanism interface {
	// Name returns the SASL mechanism name (e.g., "SCRAM-SHA-256").
	Name() string

	// Start begins the authentication exchange, returning the initial response.
	Start() ([]byte, error)

	// Next processes a challenge from the server and returns the response.
	Next(challenge []byte) ([]byte, error)

	// Completed returns true if the mechanism has completed authentication.
	Completed() bool
}

// Credentials holds authentication credentials.
type Credentials struct {
	Username       string
	Password       string
	AuthzID        string
	ChannelBinding []byte // TLS channel binding data for -PLUS variants
	CBType         string // Channel binding type (e.g., "tls-exporter")
}

// Negotiator selects and drives SASL mechanism negotiation.
type Negotiator struct {
	creds      Credentials
	mechanisms []Mechanism
}

// NewNegotiator creates a new SASL negotiator.
func NewNegotiator(creds Credentials, mechanisms ...Mechanism) *Negotiator {
	return &Negotiator{
		creds:      creds,
		mechanisms: mechanisms,
	}
}

// Select chooses the best mechanism from the server-offered list.
func (n *Negotiator) Select(offered []string) (Mechanism, error) {
	offeredSet := make(map[string]bool, len(offered))
	for _, m := range offered {
		offeredSet[m] = true
	}

	// Return the first matching mechanism (ordered by preference)
	for _, mech := range n.mechanisms {
		if offeredSet[mech.Name()] {
			return mech, nil
		}
	}
	return nil, ErrNoMechanism
}
