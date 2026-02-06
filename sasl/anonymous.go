package sasl

// Anonymous implements the ANONYMOUS SASL mechanism (RFC 4505).
type Anonymous struct {
	trace     string
	completed bool
}

// NewAnonymous creates a new ANONYMOUS mechanism.
func NewAnonymous(trace string) *Anonymous {
	return &Anonymous{trace: trace}
}

// Name returns "ANONYMOUS".
func (a *Anonymous) Name() string { return "ANONYMOUS" }

// Start returns the optional trace information.
func (a *Anonymous) Start() ([]byte, error) {
	a.completed = true
	return []byte(a.trace), nil
}

// Next is not used for ANONYMOUS.
func (a *Anonymous) Next(_ []byte) ([]byte, error) {
	return nil, nil
}

// Completed returns true after Start.
func (a *Anonymous) Completed() bool { return a.completed }
