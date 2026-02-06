package stream

import (
	"encoding/xml"
	"fmt"

	"github.com/meszmate/xmpp-go/internal/ns"
)

// Error represents an XMPP stream error (RFC 6120 §4.9).
type Error struct {
	XMLName   xml.Name `xml:"http://etherx.jabber.org/streams error"`
	Condition string
	Text      string
	AppError  *xml.Name
}

// Stream error conditions as defined in RFC 6120 §4.9.3.
const (
	ErrBadFormat              = "bad-format"
	ErrBadNamespacePrefix     = "bad-namespace-prefix"
	ErrConflict               = "conflict"
	ErrConnectionTimeout      = "connection-timeout"
	ErrHostGone               = "host-gone"
	ErrHostUnknown            = "host-unknown"
	ErrImproperAddressing     = "improper-addressing"
	ErrInternalServerError    = "internal-server-error"
	ErrInvalidFrom            = "invalid-from"
	ErrInvalidNamespace       = "invalid-namespace"
	ErrInvalidXML             = "invalid-xml"
	ErrNotAuthorized          = "not-authorized"
	ErrNotWellFormed          = "not-well-formed"
	ErrPolicyViolation        = "policy-violation"
	ErrRemoteConnectionFailed = "remote-connection-failed"
	ErrReset                  = "reset"
	ErrResourceConstraint     = "resource-constraint"
	ErrRestrictedXML          = "restricted-xml"
	ErrSeeOtherHost           = "see-other-host"
	ErrSystemShutdown         = "system-shutdown"
	ErrUndefinedCondition     = "undefined-condition"
	ErrUnsupportedEncoding    = "unsupported-encoding"
	ErrUnsupportedFeature     = "unsupported-feature"
	ErrUnsupportedStanzaType  = "unsupported-stanza-type"
	ErrUnsupportedVersion     = "unsupported-version"
)

// NewError creates a new stream error with the given condition.
func NewError(condition, text string) *Error {
	return &Error{
		Condition: condition,
		Text:      text,
	}
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Text != "" {
		return fmt.Sprintf("stream error: %s (%s)", e.Condition, e.Text)
	}
	return fmt.Sprintf("stream error: %s", e.Condition)
}

// MarshalXML implements xml.Marshaler.
func (e *Error) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Space: ns.Stream, Local: "error"}

	if err := enc.EncodeToken(start); err != nil {
		return err
	}

	// Encode condition element
	condName := xml.Name{Space: ns.Streams, Local: e.Condition}
	if err := enc.EncodeToken(xml.StartElement{Name: condName}); err != nil {
		return err
	}
	if err := enc.EncodeToken(xml.EndElement{Name: condName}); err != nil {
		return err
	}

	// Encode optional text
	if e.Text != "" {
		textName := xml.Name{Space: ns.Streams, Local: "text"}
		textStart := xml.StartElement{
			Name: textName,
			Attr: []xml.Attr{{Name: xml.Name{Local: "xml:lang"}, Value: "en"}},
		}
		if err := enc.EncodeToken(textStart); err != nil {
			return err
		}
		if err := enc.EncodeToken(xml.CharData(e.Text)); err != nil {
			return err
		}
		if err := enc.EncodeToken(xml.EndElement{Name: textName}); err != nil {
			return err
		}
	}

	return enc.EncodeToken(xml.EndElement{Name: start.Name})
}
