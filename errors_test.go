package xmpp

import (
	"testing"

	"github.com/meszmate/xmpp-go/stanza"
)

func TestErrorHelpers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		fn        func(string) *stanza.StanzaError
		wantType  string
		wantCond  string
	}{
		{"BadRequest", ErrBadRequest, stanza.ErrorTypeModify, stanza.ErrorBadRequest},
		{"Conflict", ErrConflict, stanza.ErrorTypeCancel, stanza.ErrorConflict},
		{"FeatureNotImplemented", ErrFeatureNotImplemented, stanza.ErrorTypeCancel, stanza.ErrorFeatureNotImplemented},
		{"Forbidden", ErrForbidden, stanza.ErrorTypeAuth, stanza.ErrorForbidden},
		{"ItemNotFound", ErrItemNotFound, stanza.ErrorTypeCancel, stanza.ErrorItemNotFound},
		{"NotAllowed", ErrNotAllowed, stanza.ErrorTypeCancel, stanza.ErrorNotAllowed},
		{"NotAuthorized", ErrNotAuthorized, stanza.ErrorTypeAuth, stanza.ErrorNotAuthorized},
		{"ServiceUnavailable", ErrServiceUnavailable, stanza.ErrorTypeCancel, stanza.ErrorServiceUnavailable},
		{"InternalServerError", ErrInternalServerError, stanza.ErrorTypeCancel, stanza.ErrorInternalServerError},
		{"RecipientUnavailable", ErrRecipientUnavailable, stanza.ErrorTypeWait, stanza.ErrorRecipientUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			se := tt.fn("test text")
			if se.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", se.Type, tt.wantType)
			}
			if se.Condition != tt.wantCond {
				t.Errorf("Condition = %q, want %q", se.Condition, tt.wantCond)
			}
			if se.Text != "test text" {
				t.Errorf("Text = %q, want %q", se.Text, "test text")
			}
		})
	}
}
