// Package ns defines XML namespace constants used throughout the XMPP library.
package ns

const (
	// Core XMPP namespaces (RFC 6120)
	Client  = "jabber:client"
	Server  = "jabber:server"
	Stream  = "http://etherx.jabber.org/streams"
	Streams = "urn:ietf:params:xml:ns:xmpp-streams"
	TLS     = "urn:ietf:params:xml:ns:xmpp-tls"
	SASL    = "urn:ietf:params:xml:ns:xmpp-sasl"
	Bind    = "urn:ietf:params:xml:ns:xmpp-bind"
	Session = "urn:ietf:params:xml:ns:xmpp-session"
	Stanzas = "urn:ietf:params:xml:ns:xmpp-stanzas"

	// Roster (RFC 6121)
	Roster = "jabber:iq:roster"

	// Service Discovery (XEP-0030)
	DiscoInfo  = "http://jabber.org/protocol/disco#info"
	DiscoItems = "http://jabber.org/protocol/disco#items"

	// Entity Capabilities (XEP-0115)
	Caps = "http://jabber.org/protocol/caps"

	// Data Forms (XEP-0004)
	DataForms = "jabber:x:data"

	// Multi-User Chat (XEP-0045)
	MUC      = "http://jabber.org/protocol/muc"
	MUCUser  = "http://jabber.org/protocol/muc#user"
	MUCAdmin = "http://jabber.org/protocol/muc#admin"
	MUCOwner = "http://jabber.org/protocol/muc#owner"

	// Direct MUC Invitations (XEP-0249)
	MUCInvite = "jabber:x:conference"

	// PubSub (XEP-0060)
	PubSub      = "http://jabber.org/protocol/pubsub"
	PubSubEvent = "http://jabber.org/protocol/pubsub#event"
	PubSubOwner = "http://jabber.org/protocol/pubsub#owner"

	// Message Archive Management (XEP-0313)
	MAM = "urn:xmpp:mam:2"

	// Message Carbons (XEP-0280)
	Carbons = "urn:xmpp:carbons:2"

	// Stream Management (XEP-0198)
	SM = "urn:xmpp:sm:3"

	// Message Delivery Receipts (XEP-0184)
	Receipts = "urn:xmpp:receipts"

	// Chat State Notifications (XEP-0085)
	ChatStates = "http://jabber.org/protocol/chatstates"

	// Chat Markers (XEP-0333)
	ChatMarkers = "urn:xmpp:chat-markers:0"

	// Last Message Correction (XEP-0308)
	Correction = "urn:xmpp:message-correct:0"

	// Message Retraction (XEP-0424)
	Retraction = "urn:xmpp:message-retract:1"

	// Message Reactions (XEP-0444)
	Reactions = "urn:xmpp:reactions:0"

	// Message Moderation (XEP-0425)
	Moderation = "urn:xmpp:message-moderate:1"

	// Message Styling (XEP-0393)
	Styling = "urn:xmpp:styling:0"

	// Message Processing Hints (XEP-0334)
	Hints = "urn:xmpp:hints"

	// Stanza Forwarding (XEP-0297)
	Forward = "urn:xmpp:forward:0"

	// Unique/Stable Stanza IDs (XEP-0359)
	StanzaID = "urn:xmpp:sid:0"

	// Blocking Command (XEP-0191)
	Blocking = "urn:xmpp:blocking"

	// PEP Native Bookmarks (XEP-0402)
	Bookmarks = "urn:xmpp:bookmarks:1"

	// In-Band Registration (XEP-0077)
	Register = "jabber:iq:register"

	// vcard-temp (XEP-0054)
	VCard = "vcard-temp"

	// vCard4 (XEP-0292)
	VCard4 = "urn:ietf:params:xml:ns:vcard-4.0"

	// User Avatar (XEP-0084)
	AvatarData     = "urn:xmpp:avatar:data"
	AvatarMetadata = "urn:xmpp:avatar:metadata"

	// vCard-Based Avatars (XEP-0153)
	VCardUpdate = "vcard-temp:x:update"

	// HTTP File Upload (XEP-0363)
	HTTPUpload = "urn:xmpp:http:upload:0"

	// Out of Band Data (XEP-0066)
	OOB  = "jabber:x:oob"
	OOB2 = "jabber:iq:oob"

	// In-Band Bytestreams (XEP-0047)
	IBB = "http://jabber.org/protocol/ibb"

	// SOCKS5 Bytestreams (XEP-0065)
	SOCKS5 = "http://jabber.org/protocol/bytestreams"

	// Jingle (XEP-0166)
	Jingle = "urn:xmpp:jingle:1"

	// Jingle File Transfer (XEP-0234)
	JingleFT = "urn:xmpp:jingle:apps:file-transfer:5"

	// Jingle RTP Sessions (XEP-0167)
	JingleRTP = "urn:xmpp:jingle:apps:rtp:1"

	// Jingle ICE-UDP Transport (XEP-0176)
	JingleICEUDP = "urn:xmpp:jingle:transports:ice-udp:1"

	// Jingle Raw UDP Transport (XEP-0177)
	JingleRawUDP = "urn:xmpp:jingle:transports:raw-udp:1"

	// DTLS-SRTP in Jingle (XEP-0320)
	JingleDTLS = "urn:xmpp:jingle:apps:dtls:0"

	// Jingle Message Initiation (XEP-0353)
	JingleMI = "urn:xmpp:jingle-message:0"

	// OMEMO Encryption (XEP-0384)
	OMEMO = "urn:xmpp:omemo:2"

	// Explicit Message Encryption (XEP-0380)
	EME = "urn:xmpp:eme:0"

	// OMEMO Media Sharing (XEP-0454)
	OMEMOMedia = "urn:xmpp:encrypted:0"

	// Ad-Hoc Commands (XEP-0050)
	Commands = "http://jabber.org/protocol/commands"

	// Client State Indication (XEP-0352)
	CSI = "urn:xmpp:csi:0"

	// Push Notifications (XEP-0357)
	Push = "urn:xmpp:push:0"

	// Last Activity (XEP-0012)
	LastActivity = "jabber:iq:last"

	// Result Set Management (XEP-0059)
	RSM = "http://jabber.org/protocol/rsm"

	// Cryptographic Hash Functions (XEP-0300)
	Hashes = "urn:xmpp:hashes:2"

	// XMPP Ping (XEP-0199)
	Ping = "urn:xmpp:ping"

	// Software Version (XEP-0092)
	Version = "jabber:iq:version"

	// Entity Time (XEP-0202)
	Time = "urn:xmpp:time"

	// Delayed Delivery (XEP-0203)
	Delay = "urn:xmpp:delay"

	// Bits of Binary (XEP-0231)
	BoB = "urn:xmpp:bob"

	// SASL2 (XEP-0388)
	SASL2 = "urn:xmpp:sasl:2"

	// SASL Channel-Binding (XEP-0440)
	SASLCBind = "urn:xmpp:sasl-cb:0"

	// FAST (XEP-0484)
	FAST = "urn:xmpp:fast:0"

	// Bind 2 (XEP-0386)
	Bind2 = "urn:xmpp:bind:0"

	// MIX (XEP-0369)
	MIXCore     = "urn:xmpp:mix:core:1"
	MIXNodes    = "urn:xmpp:mix:nodes:1"
	MIXPAM      = "urn:xmpp:mix:pam:2"
	MIXAdmin    = "urn:xmpp:mix:admin:0"
	MIXAnon     = "urn:xmpp:mix:anon:0"
	MIXPresence = "urn:xmpp:mix:presence:0"

	// External Service Discovery (XEP-0215)
	ExtDisco = "urn:xmpp:extdisco:2"

	// Server Dialback (XEP-0220)
	Dialback = "jabber:server:dialback"

	// Bidirectional S2S (XEP-0288)
	BidiS2S = "urn:xmpp:bidi"

	// Component (XEP-0114)
	Component       = "jabber:component:accept"
	ComponentSecret = "jabber:component:connect"

	// WebSocket framing (RFC 7395)
	Framing = "urn:ietf:params:xml:ns:xmpp-framing"

	// BOSH (XEP-0124/0206)
	BOSH     = "http://jabber.org/protocol/httpbind"
	BOSHXmpp = "urn:xmpp:xbosh"

	// Stateless File Sharing (XEP-0446/0447/0448)
	FileMetadata = "urn:xmpp:file:metadata:0"
	SFS          = "urn:xmpp:sfs:0"
	SFSEncrypted = "urn:xmpp:esfs:0"
)
