package xmpp

import (
	"testing"

	"github.com/meszmate/xmpp-go/jid"
)

func TestLocalRouterFullJID(t *testing.T) {
	r := newLocalRouter()
	s1 := &Session{}
	full := jid.MustParse("alice@example.com/phone")
	r.register(full, s1)

	got := r.targets(full)
	if len(got) != 1 || got[0] != s1 {
		t.Fatalf("targets(full) = %v, want [s1]", got)
	}
	// Bare JID should fan out to the one resource.
	got = r.targets(jid.MustParse("alice@example.com"))
	if len(got) != 1 || got[0] != s1 {
		t.Fatalf("targets(bare) = %v, want [s1]", got)
	}
	if !r.online(jid.MustParse("alice@example.com")) {
		t.Error("online(alice) should be true")
	}

	r.unregister(full)
	if len(r.targets(full)) != 0 {
		t.Error("targets after unregister should be empty")
	}
	if r.online(jid.MustParse("alice@example.com")) {
		t.Error("online(alice) should be false after unregister")
	}
}

func TestLocalRouterMultipleResources(t *testing.T) {
	r := newLocalRouter()
	s1, s2 := &Session{}, &Session{}
	r.register(jid.MustParse("alice@example.com/phone"), s1)
	r.register(jid.MustParse("alice@example.com/laptop"), s2)

	// Bare JID fans out to both.
	got := r.targets(jid.MustParse("alice@example.com"))
	if len(got) != 2 {
		t.Fatalf("bare fan-out = %d sessions, want 2", len(got))
	}
	// Full JID targets exactly one.
	got = r.targets(jid.MustParse("alice@example.com/phone"))
	if len(got) != 1 || got[0] != s1 {
		t.Fatalf("full JID target wrong: %v", got)
	}
	// Removing one resource leaves the other.
	r.unregister(jid.MustParse("alice@example.com/phone"))
	got = r.targets(jid.MustParse("alice@example.com"))
	if len(got) != 1 || got[0] != s2 {
		t.Fatalf("after removing phone, bare = %v, want [laptop]", got)
	}
}

func TestLocalRouterUnknownAndZero(t *testing.T) {
	r := newLocalRouter()
	if got := r.targets(jid.MustParse("nobody@example.com")); got != nil {
		t.Errorf("targets(unknown) = %v, want nil", got)
	}
	if got := r.targets(jid.JID{}); got != nil {
		t.Errorf("targets(zero) = %v, want nil", got)
	}
	// Register/unregister of a zero JID must be a no-op (no panic).
	r.register(jid.JID{}, &Session{})
	r.unregister(jid.JID{})
}
