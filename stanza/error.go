package stanza

import (
	"encoding/xml"
	"fmt"

	"github.com/meszmate/xmpp-go/internal/ns"
)

// Error type constants (RFC 6120 §8.3.2).
const (
	ErrorTypeAuth     = "auth"
	ErrorTypeCancel   = "cancel"
	ErrorTypeContinue = "continue"
	ErrorTypeModify   = "modify"
	ErrorTypeWait     = "wait"
)

// Error condition constants (RFC 6120 §8.3.3).
const (
	ErrorBadRequest            = "bad-request"
	ErrorConflict              = "conflict"
	ErrorFeatureNotImplemented = "feature-not-implemented"
	ErrorForbidden             = "forbidden"
	ErrorGone                  = "gone"
	ErrorInternalServerError   = "internal-server-error"
	ErrorItemNotFound          = "item-not-found"
	ErrorJIDMalformed          = "jid-malformed"
	ErrorNotAcceptable         = "not-acceptable"
	ErrorNotAllowed            = "not-allowed"
	ErrorNotAuthorized         = "not-authorized"
	ErrorPolicyViolation       = "policy-violation"
	ErrorRecipientUnavailable  = "recipient-unavailable"
	ErrorRedirect              = "redirect"
	ErrorRegistrationRequired  = "registration-required"
	ErrorRemoteServerNotFound  = "remote-server-not-found"
	ErrorRemoteServerTimeout   = "remote-server-timeout"
	ErrorResourceConstraint    = "resource-constraint"
	ErrorServiceUnavailable    = "service-unavailable"
	ErrorSubscriptionRequired  = "subscription-required"
	ErrorUndefinedCondition    = "undefined-condition"
	ErrorUnexpectedRequest     = "unexpected-request"
)

// StanzaError represents an XMPP stanza error.
type StanzaError struct {
	XMLName   xml.Name `xml:"error"`
	Type      string   `xml:"type,attr"`
	By        string   `xml:"by,attr,omitempty"`
	Condition string   `xml:"-"`
	Text      string   `xml:"-"`
}

// NewStanzaError creates a new StanzaError.
func NewStanzaError(typ, condition, text string) *StanzaError {
	return &StanzaError{
		Type:      typ,
		Condition: condition,
		Text:      text,
	}
}

// Error implements the error interface.
func (e *StanzaError) Error() string {
	if e.Text != "" {
		return fmt.Sprintf("stanza error: %s (%s: %s)", e.Condition, e.Type, e.Text)
	}
	return fmt.Sprintf("stanza error: %s (%s)", e.Condition, e.Type)
}

// MarshalXML implements xml.Marshaler.
func (e *StanzaError) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "error"}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "type"}, Value: e.Type},
	}
	if e.By != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "by"}, Value: e.By})
	}

	if err := enc.EncodeToken(start); err != nil {
		return err
	}

	condName := xml.Name{Space: ns.Stanzas, Local: e.Condition}
	if err := enc.EncodeToken(xml.StartElement{Name: condName}); err != nil {
		return err
	}
	if err := enc.EncodeToken(xml.EndElement{Name: condName}); err != nil {
		return err
	}

	if e.Text != "" {
		textName := xml.Name{Space: ns.Stanzas, Local: "text"}
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
