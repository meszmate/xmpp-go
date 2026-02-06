// Package form implements XEP-0004 Data Forms.
package form

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "form"

// Form type constants.
const (
	TypeForm   = "form"
	TypeSubmit = "submit"
	TypeCancel = "cancel"
	TypeResult = "result"
)

// Field type constants.
const (
	FieldBoolean    = "boolean"
	FieldFixed      = "fixed"
	FieldHidden     = "hidden"
	FieldJIDMulti   = "jid-multi"
	FieldJIDSingle  = "jid-single"
	FieldListMulti  = "list-multi"
	FieldListSingle = "list-single"
	FieldTextMulti  = "text-multi"
	FieldTextPrivate = "text-private"
	FieldTextSingle = "text-single"
)

// Form represents an XEP-0004 data form.
type Form struct {
	XMLName      xml.Name `xml:"jabber:x:data x"`
	Type         string   `xml:"type,attr"`
	Title        string   `xml:"title,omitempty"`
	Instructions []string `xml:"instructions,omitempty"`
	Fields       []Field  `xml:"field"`
	Reported     *Reported `xml:"reported,omitempty"`
	Items        []FormItem `xml:"item,omitempty"`
}

// Field represents a form field.
type Field struct {
	XMLName  xml.Name `xml:"field"`
	Var      string   `xml:"var,attr,omitempty"`
	Type     string   `xml:"type,attr,omitempty"`
	Label    string   `xml:"label,attr,omitempty"`
	Required bool     `xml:"-"`
	Desc     string   `xml:"desc,omitempty"`
	Values   []string `xml:"value,omitempty"`
	Options  []Option `xml:"option,omitempty"`
}

// Option represents a field option.
type Option struct {
	XMLName xml.Name `xml:"option"`
	Label   string   `xml:"label,attr,omitempty"`
	Value   string   `xml:"value"`
}

// Reported represents reported fields in a result form.
type Reported struct {
	XMLName xml.Name `xml:"reported"`
	Fields  []Field  `xml:"field"`
}

// FormItem represents a result item.
type FormItem struct {
	XMLName xml.Name `xml:"item"`
	Fields  []Field  `xml:"field"`
}

// Plugin implements XEP-0004.
type Plugin struct {
	params plugin.InitParams
}

// New creates a new data forms plugin.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }

func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	return nil
}

func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

// NewForm creates a new form with the given type and title.
func NewForm(typ, title string) *Form {
	return &Form{Type: typ, Title: title}
}

// AddField adds a field to the form.
func (f *Form) AddField(field Field) {
	f.Fields = append(f.Fields, field)
}

// GetField returns the field with the given var.
func (f *Form) GetField(varName string) *Field {
	for i := range f.Fields {
		if f.Fields[i].Var == varName {
			return &f.Fields[i]
		}
	}
	return nil
}

// GetValue returns the first value of a field.
func (f *Form) GetValue(varName string) string {
	field := f.GetField(varName)
	if field != nil && len(field.Values) > 0 {
		return field.Values[0]
	}
	return ""
}

// GetValues returns all values of a field.
func (f *Form) GetValues(varName string) []string {
	field := f.GetField(varName)
	if field != nil {
		return field.Values
	}
	return nil
}

func init() {
	_ = ns.DataForms
}
