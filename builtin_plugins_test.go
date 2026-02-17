package xmpp_test

import (
	"context"
	"testing"

	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/plugins/avatar"
	"github.com/meszmate/xmpp-go/plugins/blocking"
	"github.com/meszmate/xmpp-go/plugins/bob"
	"github.com/meszmate/xmpp-go/plugins/bookmarks"
	"github.com/meszmate/xmpp-go/plugins/caps"
	"github.com/meszmate/xmpp-go/plugins/carbons"
	"github.com/meszmate/xmpp-go/plugins/chatmarkers"
	"github.com/meszmate/xmpp-go/plugins/chatstates"
	"github.com/meszmate/xmpp-go/plugins/commands"
	"github.com/meszmate/xmpp-go/plugins/correction"
	"github.com/meszmate/xmpp-go/plugins/csi"
	"github.com/meszmate/xmpp-go/plugins/delay"
	"github.com/meszmate/xmpp-go/plugins/dialback"
	"github.com/meszmate/xmpp-go/plugins/disco"
	"github.com/meszmate/xmpp-go/plugins/extdisco"
	"github.com/meszmate/xmpp-go/plugins/filetransfer"
	"github.com/meszmate/xmpp-go/plugins/form"
	"github.com/meszmate/xmpp-go/plugins/forward"
	"github.com/meszmate/xmpp-go/plugins/hash"
	"github.com/meszmate/xmpp-go/plugins/hints"
	"github.com/meszmate/xmpp-go/plugins/ibb"
	"github.com/meszmate/xmpp-go/plugins/jingle"
	"github.com/meszmate/xmpp-go/plugins/lastactivity"
	"github.com/meszmate/xmpp-go/plugins/mam"
	"github.com/meszmate/xmpp-go/plugins/mix"
	"github.com/meszmate/xmpp-go/plugins/moderation"
	"github.com/meszmate/xmpp-go/plugins/muc"
	"github.com/meszmate/xmpp-go/plugins/omemo"
	"github.com/meszmate/xmpp-go/plugins/oob"
	"github.com/meszmate/xmpp-go/plugins/ping"
	"github.com/meszmate/xmpp-go/plugins/presence"
	"github.com/meszmate/xmpp-go/plugins/pubsub"
	"github.com/meszmate/xmpp-go/plugins/push"
	"github.com/meszmate/xmpp-go/plugins/reactions"
	"github.com/meszmate/xmpp-go/plugins/receipts"
	"github.com/meszmate/xmpp-go/plugins/register"
	"github.com/meszmate/xmpp-go/plugins/retraction"
	"github.com/meszmate/xmpp-go/plugins/roster"
	"github.com/meszmate/xmpp-go/plugins/rsm"
	"github.com/meszmate/xmpp-go/plugins/sasl2"
	"github.com/meszmate/xmpp-go/plugins/sm"
	"github.com/meszmate/xmpp-go/plugins/socks5"
	"github.com/meszmate/xmpp-go/plugins/stanzaid"
	"github.com/meszmate/xmpp-go/plugins/styling"
	"github.com/meszmate/xmpp-go/plugins/time"
	"github.com/meszmate/xmpp-go/plugins/upload"
	"github.com/meszmate/xmpp-go/plugins/vcard"
	"github.com/meszmate/xmpp-go/plugins/version"
	"github.com/meszmate/xmpp-go/storage/memory"
)

func TestBuiltinPluginsInitializeAndClose(t *testing.T) {
	mgr := plugin.NewManager()
	all := []plugin.Plugin{
		avatar.New(),
		blocking.New(),
		bob.New(),
		bookmarks.New(),
		caps.New("https://example.com/client"),
		carbons.New(),
		chatmarkers.New(),
		chatstates.New(),
		commands.New(),
		correction.New(),
		csi.New(),
		delay.New(),
		dialback.New(),
		disco.New(),
		extdisco.New(),
		filetransfer.New(),
		form.New(),
		forward.New(),
		hash.New(),
		hints.New(),
		ibb.New(),
		jingle.New(),
		lastactivity.New(),
		mam.New(),
		mix.New(),
		moderation.New(),
		muc.New(),
		oob.New(),
		omemo.New(123456),
		ping.New(),
		presence.New(),
		pubsub.New(),
		push.New(),
		reactions.New(),
		receipts.New(),
		register.New(),
		retraction.New(),
		roster.New(),
		rsm.New(),
		sasl2.New(),
		sm.New(),
		socks5.New(),
		stanzaid.New(),
		styling.New(),
		time.New(),
		upload.New(),
		vcard.New(),
		version.New("xmpp-go", "test"),
	}

	for _, p := range all {
		if err := mgr.Register(p); err != nil {
			t.Fatalf("register %q: %v", p.Name(), err)
		}
	}

	params := plugin.InitParams{
		SendRaw: func(context.Context, []byte) error { return nil },
		SendElement: func(context.Context, any) error {
			return nil
		},
		State:     func() uint32 { return 0 },
		LocalJID:  func() string { return "alice@example.com" },
		RemoteJID: func() string { return "bob@example.com" },
		Storage:   memory.New(),
	}

	if err := mgr.Initialize(context.Background(), params); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if err := mgr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
