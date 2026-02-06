// Package caps implements XEP-0115 Entity Capabilities.
package caps

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"sort"
	"strings"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/plugins/disco"
)

const Name = "caps"

// Caps represents an entity capabilities element.
type Caps struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/caps c"`
	Hash    string   `xml:"hash,attr"`
	Node    string   `xml:"node,attr"`
	Ver     string   `xml:"ver,attr"`
}

// Plugin implements XEP-0115.
type Plugin struct {
	node   string
	params plugin.InitParams
}

// New creates a new caps plugin with the given node URI.
func New(node string) *Plugin {
	return &Plugin{node: node}
}

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }

func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	return nil
}

func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return []string{disco.Name} }

// Ver computes the verification string from disco info.
func (p *Plugin) Ver(info disco.InfoQuery) string {
	var s strings.Builder

	// Sort identities
	ids := make([]disco.Identity, len(info.Identities))
	copy(ids, info.Identities)
	sort.Slice(ids, func(i, j int) bool {
		a := ids[i].Category + "/" + ids[i].Type + "/" + ids[i].Lang + "/" + ids[i].Name
		b := ids[j].Category + "/" + ids[j].Type + "/" + ids[j].Lang + "/" + ids[j].Name
		return a < b
	})

	for _, id := range ids {
		s.WriteString(id.Category + "/" + id.Type + "/" + id.Lang + "/" + id.Name + "<")
	}

	// Sort features
	feats := make([]string, len(info.Features))
	for i, f := range info.Features {
		feats[i] = f.Var
	}
	sort.Strings(feats)

	for _, f := range feats {
		s.WriteString(f + "<")
	}

	h := sha1.Sum([]byte(s.String()))
	return base64.StdEncoding.EncodeToString(h[:])
}

// Generate creates a Caps element from the current disco info.
func (p *Plugin) Generate(info disco.InfoQuery) Caps {
	return Caps{
		Hash: "sha-1",
		Node: p.node,
		Ver:  p.Ver(info),
	}
}

func init() {
	_ = ns.Caps // ensure ns import is used
}
