// Package bookmarks implements XEP-0402 PEP Native Bookmarks.
package bookmarks

import (
	"context"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/storage"
)

const Name = "bookmarks"

// PEP node for bookmarks.
const Node = "urn:xmpp:bookmarks:1"

type Conference struct {
	XMLName    xml.Name    `xml:"urn:xmpp:bookmarks:1 conference"`
	Autojoin   bool        `xml:"autojoin,attr,omitempty"`
	Name       string      `xml:"name,attr,omitempty"`
	Nick       string      `xml:"nick,omitempty"`
	Password   string      `xml:"password,omitempty"`
	Extensions []Extension `xml:"extensions,omitempty"`
}

type Extension struct {
	XMLName xml.Name
	Inner   []byte `xml:",innerxml"`
}

type Plugin struct {
	store  storage.BookmarkStore
	params plugin.InitParams
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }
func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	if params.Storage != nil {
		p.store = params.Storage.BookmarkStore()
	}
	return nil
}
func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

// Set adds or updates a bookmark. Returns nil if no store is configured.
func (p *Plugin) Set(ctx context.Context, bm *storage.Bookmark) error {
	if p.store == nil {
		return nil
	}
	return p.store.SetBookmark(ctx, bm)
}

// Get retrieves a bookmark. Returns nil if no store is configured.
func (p *Plugin) Get(ctx context.Context, userJID, roomJID string) (*storage.Bookmark, error) {
	if p.store == nil {
		return nil, nil
	}
	return p.store.GetBookmark(ctx, userJID, roomJID)
}

// List retrieves all bookmarks for a user. Returns nil if no store is configured.
func (p *Plugin) List(ctx context.Context, userJID string) ([]*storage.Bookmark, error) {
	if p.store == nil {
		return nil, nil
	}
	return p.store.GetBookmarks(ctx, userJID)
}

// Delete removes a bookmark. Returns nil if no store is configured.
func (p *Plugin) Delete(ctx context.Context, userJID, roomJID string) error {
	if p.store == nil {
		return nil
	}
	return p.store.DeleteBookmark(ctx, userJID, roomJID)
}

func init() { _ = ns.Bookmarks }
