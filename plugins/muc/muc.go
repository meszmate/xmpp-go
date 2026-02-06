// Package muc implements XEP-0045 Multi-User Chat and XEP-0249 Direct MUC Invitations.
package muc

import (
	"context"
	"encoding/xml"
	"sync"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "muc"

// Affiliations
const (
	AffOwner   = "owner"
	AffAdmin   = "admin"
	AffMember  = "member"
	AffOutcast = "outcast"
	AffNone    = "none"
)

// Roles
const (
	RoleModerator   = "moderator"
	RoleParticipant = "participant"
	RoleVisitor     = "visitor"
	RoleNone        = "none"
)

type MUC struct {
	XMLName  xml.Name `xml:"http://jabber.org/protocol/muc x"`
	History  *History `xml:"history,omitempty"`
	Password string   `xml:"password,omitempty"`
}

type History struct {
	XMLName    xml.Name `xml:"history"`
	MaxChars   *int     `xml:"maxchars,attr,omitempty"`
	MaxStanzas *int     `xml:"maxstanzas,attr,omitempty"`
	Seconds    *int     `xml:"seconds,attr,omitempty"`
	Since      string   `xml:"since,attr,omitempty"`
}

type UserX struct {
	XMLName xml.Name    `xml:"http://jabber.org/protocol/muc#user x"`
	Items   []UserItem  `xml:"item"`
	Status  []Status    `xml:"status"`
	Invite  []Invite    `xml:"invite"`
	Decline *Decline    `xml:"decline,omitempty"`
}

type UserItem struct {
	XMLName     xml.Name `xml:"item"`
	Affiliation string   `xml:"affiliation,attr,omitempty"`
	Role        string   `xml:"role,attr,omitempty"`
	JID         string   `xml:"jid,attr,omitempty"`
	Nick        string   `xml:"nick,attr,omitempty"`
	Reason      string   `xml:"reason,omitempty"`
}

type Status struct {
	XMLName xml.Name `xml:"status"`
	Code    int      `xml:"code,attr"`
}

type Invite struct {
	XMLName xml.Name `xml:"invite"`
	From    string   `xml:"from,attr,omitempty"`
	To      string   `xml:"to,attr,omitempty"`
	Reason  string   `xml:"reason,omitempty"`
}

type Decline struct {
	XMLName xml.Name `xml:"decline"`
	From    string   `xml:"from,attr,omitempty"`
	To      string   `xml:"to,attr,omitempty"`
	Reason  string   `xml:"reason,omitempty"`
}

type AdminQuery struct {
	XMLName xml.Name   `xml:"http://jabber.org/protocol/muc#admin query"`
	Items   []UserItem `xml:"item"`
}

type OwnerQuery struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/muc#owner query"`
	Form    []byte   `xml:",innerxml"`
}

// DirectInvite represents XEP-0249.
type DirectInvite struct {
	XMLName  xml.Name `xml:"jabber:x:conference x"`
	JID      string   `xml:"jid,attr"`
	Password string   `xml:"password,attr,omitempty"`
	Reason   string   `xml:"reason,attr,omitempty"`
}

type Room struct {
	JID     string
	Nick    string
	Joined  bool
}

type Plugin struct {
	mu     sync.RWMutex
	rooms  map[string]*Room
	params plugin.InitParams
}

func New() *Plugin {
	return &Plugin{rooms: make(map[string]*Room)}
}

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }
func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	return nil
}
func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

func (p *Plugin) JoinRoom(roomJID, nick string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rooms[roomJID] = &Room{JID: roomJID, Nick: nick, Joined: true}
}

func (p *Plugin) LeaveRoom(roomJID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.rooms, roomJID)
}

func (p *Plugin) GetRoom(roomJID string) (*Room, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	r, ok := p.rooms[roomJID]
	return r, ok
}

func (p *Plugin) Rooms() []*Room {
	p.mu.RLock()
	defer p.mu.RUnlock()
	rooms := make([]*Room, 0, len(p.rooms))
	for _, r := range p.rooms {
		rooms = append(rooms, r)
	}
	return rooms
}

func init() {
	_ = ns.MUC
	_ = ns.MUCUser
	_ = ns.MUCAdmin
	_ = ns.MUCOwner
	_ = ns.MUCInvite
}
