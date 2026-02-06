// Package time implements XEP-0082 Date/Time Profiles and XEP-0202 Entity Time.
package time

import (
	"context"
	"encoding/xml"
	gotime "time"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "time"

// Time represents an XEP-0202 entity time response.
type Time struct {
	XMLName xml.Name `xml:"urn:xmpp:time time"`
	TZO     string   `xml:"tzo"`
	UTC     string   `xml:"utc"`
}

// Plugin implements XEP-0082/0202.
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

// Now returns the current entity time.
func (p *Plugin) Now() Time {
	now := gotime.Now()
	return Time{
		TZO: now.Format("-07:00"),
		UTC: now.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// FormatDateTime formats a time per XEP-0082 DateTime profile.
func FormatDateTime(t gotime.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

// ParseDateTime parses an XEP-0082 DateTime string.
func ParseDateTime(s string) (gotime.Time, error) {
	return gotime.Parse("2006-01-02T15:04:05Z", s)
}

// FormatDate formats a time per XEP-0082 Date profile.
func FormatDate(t gotime.Time) string {
	return t.Format("2006-01-02")
}

// FormatTime formats a time per XEP-0082 Time profile.
func FormatTime(t gotime.Time) string {
	return t.UTC().Format("15:04:05Z")
}

func init() { _ = ns.Time }
