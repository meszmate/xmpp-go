package sasl

// External implements the EXTERNAL SASL mechanism.
type External struct {
	authzID   string
	completed bool
}

// NewExternal creates a new EXTERNAL mechanism.
func NewExternal(authzID string) *External {
	return &External{authzID: authzID}
}

// Name returns "EXTERNAL".
func (e *External) Name() string { return "EXTERNAL" }

// Start returns the authorization identity.
func (e *External) Start() ([]byte, error) {
	e.completed = true
	if e.authzID != "" {
		return []byte(e.authzID), nil
	}
	return []byte{}, nil
}

// Next is not used for EXTERNAL.
func (e *External) Next(_ []byte) ([]byte, error) {
	return nil, nil
}

// Completed returns true after Start.
func (e *External) Completed() bool { return e.completed }
