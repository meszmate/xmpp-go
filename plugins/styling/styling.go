// Package styling implements XEP-0393 Message Styling.
package styling

import (
	"context"
	"strings"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "styling"

// Span types.
const (
	SpanEmphasis      = "*"
	SpanStrong        = "**"
	SpanStrikethrough = "~"
	SpanPreformatted  = "`"
)

// Span represents a styled span of text.
type Span struct {
	Type  string
	Start int
	End   int
	Text  string
}

// Plugin implements XEP-0393.
type Plugin struct {
	params plugin.InitParams
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }
func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	return nil
}
func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

// IsPreformattedBlock returns true if the line starts a preformatted block.
func IsPreformattedBlock(line string) bool {
	return strings.HasPrefix(line, "```")
}

// IsQuoteLine returns true if the line is a block quote.
func IsQuoteLine(line string) bool {
	return strings.HasPrefix(line, "> ")
}

func init() { _ = ns.Styling }
