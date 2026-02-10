package stream

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
)

func TestNewStreamError(t *testing.T) {
	t.Parallel()
	conditions := []string{
		ErrBadFormat, ErrBadNamespacePrefix, ErrConflict,
		ErrConnectionTimeout, ErrHostGone, ErrHostUnknown,
		ErrImproperAddressing, ErrInternalServerError, ErrInvalidFrom,
		ErrInvalidNamespace, ErrInvalidXML, ErrNotAuthorized,
		ErrNotWellFormed, ErrPolicyViolation, ErrRemoteConnectionFailed,
		ErrReset, ErrResourceConstraint, ErrRestrictedXML,
		ErrSeeOtherHost, ErrSystemShutdown, ErrUndefinedCondition,
		ErrUnsupportedEncoding, ErrUnsupportedFeature,
		ErrUnsupportedStanzaType, ErrUnsupportedVersion,
	}
	for _, cond := range conditions {
		t.Run(cond, func(t *testing.T) {
			t.Parallel()
			e := NewError(cond, "")
			if e.Condition != cond {
				t.Errorf("Condition = %q, want %q", e.Condition, cond)
			}
		})
	}
}

func TestStreamErrorString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			"without text",
			NewError(ErrNotAuthorized, ""),
			"stream error: not-authorized",
		},
		{
			"with text",
			NewError(ErrHostUnknown, "no such host"),
			"stream error: host-unknown (no such host)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStreamErrorMarshalXML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		err       *Error
		wantCond  string
		wantText  string
	}{
		{
			"condition only",
			NewError(ErrBadFormat, ""),
			"bad-format",
			"",
		},
		{
			"with text",
			NewError(ErrInvalidXML, "parse failure"),
			"invalid-xml",
			"parse failure",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			enc := xml.NewEncoder(&buf)
			if err := enc.Encode(tt.err); err != nil {
				t.Fatalf("Encode: %v", err)
			}
			out := buf.String()
			if !strings.Contains(out, tt.wantCond) {
				t.Errorf("missing condition %q in: %s", tt.wantCond, out)
			}
			if tt.wantText != "" && !strings.Contains(out, tt.wantText) {
				t.Errorf("missing text %q in: %s", tt.wantText, out)
			}
		})
	}
}
