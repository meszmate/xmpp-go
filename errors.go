package xmpp

import (
	"github.com/meszmate/xmpp-go/stanza"
)

// Common stanza errors as convenience constructors.

func ErrBadRequest(text string) *stanza.StanzaError {
	return stanza.NewStanzaError(stanza.ErrorTypeModify, stanza.ErrorBadRequest, text)
}

func ErrConflict(text string) *stanza.StanzaError {
	return stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorConflict, text)
}

func ErrFeatureNotImplemented(text string) *stanza.StanzaError {
	return stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorFeatureNotImplemented, text)
}

func ErrForbidden(text string) *stanza.StanzaError {
	return stanza.NewStanzaError(stanza.ErrorTypeAuth, stanza.ErrorForbidden, text)
}

func ErrItemNotFound(text string) *stanza.StanzaError {
	return stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorItemNotFound, text)
}

func ErrNotAllowed(text string) *stanza.StanzaError {
	return stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorNotAllowed, text)
}

func ErrNotAuthorized(text string) *stanza.StanzaError {
	return stanza.NewStanzaError(stanza.ErrorTypeAuth, stanza.ErrorNotAuthorized, text)
}

func ErrServiceUnavailable(text string) *stanza.StanzaError {
	return stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorServiceUnavailable, text)
}

func ErrInternalServerError(text string) *stanza.StanzaError {
	return stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorInternalServerError, text)
}

func ErrRecipientUnavailable(text string) *stanza.StanzaError {
	return stanza.NewStanzaError(stanza.ErrorTypeWait, stanza.ErrorRecipientUnavailable, text)
}
