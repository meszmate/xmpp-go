package sasl

import "fmt"

// Plain implements the PLAIN SASL mechanism (RFC 4616).
type Plain struct {
	creds     Credentials
	completed bool
}

// NewPlain creates a new PLAIN mechanism.
func NewPlain(creds Credentials) *Plain {
	return &Plain{creds: creds}
}

// Name returns "PLAIN".
func (p *Plain) Name() string { return "PLAIN" }

// Start returns the initial PLAIN response: [authzid]\0authcid\0passwd.
func (p *Plain) Start() ([]byte, error) {
	resp := fmt.Sprintf("%s\x00%s\x00%s", p.creds.AuthzID, p.creds.Username, p.creds.Password)
	p.completed = true
	return []byte(resp), nil
}

// Next is not used for PLAIN (single-step mechanism).
func (p *Plain) Next(_ []byte) ([]byte, error) {
	return nil, nil
}

// Completed returns true after Start.
func (p *Plain) Completed() bool { return p.completed }
