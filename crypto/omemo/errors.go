package omemo

import "errors"

var (
	ErrNoSession        = errors.New("omemo: no session exists for address")
	ErrInvalidSignature = errors.New("omemo: invalid signature")
	ErrInvalidMessage   = errors.New("omemo: invalid message")
	ErrDuplicateMessage = errors.New("omemo: duplicate message")
	ErrUntrustedIdentity = errors.New("omemo: untrusted identity key")
	ErrNoPreKey         = errors.New("omemo: no pre-key available")
	ErrInvalidKeyLength = errors.New("omemo: invalid key length")
	ErrSkippedKeyLimit  = errors.New("omemo: too many skipped message keys")
)
