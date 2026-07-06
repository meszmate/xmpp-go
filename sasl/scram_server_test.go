package sasl

import (
	"errors"
	"testing"
)

func lookup(users map[string]string) PasswordLookup {
	return func(u string) (string, bool) {
		p, ok := users[u]
		return p, ok
	}
}

// runSCRAM drives a full client<->server SCRAM exchange and returns whether both
// sides completed successfully.
func runSCRAM(t *testing.T, mech string, client *SCRAM, server *SCRAMServer) error {
	t.Helper()

	clientFirst, err := client.Start()
	if err != nil {
		return err
	}
	serverFirst, err := server.Start(clientFirst)
	if err != nil {
		return err
	}
	clientFinal, err := client.Next(serverFirst)
	if err != nil {
		return err
	}
	serverFinal, err := server.Finish(clientFinal)
	if err != nil {
		return err
	}
	// Client verifies the server signature.
	if _, err := client.Next(serverFinal); err != nil {
		return err
	}
	if !client.Completed() {
		t.Errorf("%s: client not completed", mech)
	}
	if !server.Completed() {
		t.Errorf("%s: server not completed", mech)
	}
	return nil
}

func TestSCRAMRoundTrip(t *testing.T) {
	users := map[string]string{"alice": "s3cret", "bob": "hunter2"}

	cases := []struct {
		mech   string
		client func(Credentials) *SCRAM
		server func(PasswordLookup) *SCRAMServer
	}{
		{"SCRAM-SHA-1", NewSCRAMSHA1, NewSCRAMServerSHA1},
		{"SCRAM-SHA-256", NewSCRAMSHA256, NewSCRAMServerSHA256},
		{"SCRAM-SHA-512", NewSCRAMSHA512, NewSCRAMServerSHA512},
	}

	for _, tc := range cases {
		t.Run(tc.mech, func(t *testing.T) {
			creds := Credentials{Username: "alice", Password: "s3cret"}
			client := tc.client(creds)
			server := tc.server(lookup(users))
			if err := runSCRAM(t, tc.mech, client, server); err != nil {
				t.Fatalf("%s round trip failed: %v", tc.mech, err)
			}
			if server.Username() != "alice" {
				t.Errorf("server parsed username %q, want alice", server.Username())
			}
		})
	}
}

// TestSCRAMPlusRoundTrip verifies a full -PLUS exchange with matching channel
// binding succeeds.
func TestSCRAMPlusRoundTrip(t *testing.T) {
	users := map[string]string{"alice": "s3cret"}
	cb := []byte("tls-channel-binding-data-example")

	creds := Credentials{Username: "alice", Password: "s3cret", CBType: "tls-server-end-point", ChannelBinding: cb}
	client := NewSCRAMSHA256Plus(creds)
	server := NewSCRAMServer("SCRAM-SHA-256-PLUS", lookup(users))
	if server == nil {
		t.Fatal("NewSCRAMServer returned nil for -PLUS")
	}
	server.SetChannelBinding(cb)

	if err := runSCRAM(t, "SCRAM-SHA-256-PLUS", client, server); err != nil {
		t.Fatalf("-PLUS round trip failed: %v", err)
	}
	if server.Username() != "alice" {
		t.Errorf("username = %q", server.Username())
	}
}

// TestSCRAMServerPlusBindingMismatch is the core security test: if the client's
// channel binding does not match the server's TLS channel, the server rejects
// the exchange (defeating a MITM that relays SCRAM over a different TLS channel).
func TestSCRAMServerPlusBindingMismatch(t *testing.T) {
	users := map[string]string{"alice": "s3cret"}
	client := NewSCRAMSHA256Plus(Credentials{
		Username: "alice", Password: "s3cret",
		CBType: "tls-server-end-point", ChannelBinding: []byte("CLIENT-sees-channel-A"),
	})
	server := NewSCRAMServer("SCRAM-SHA-256-PLUS", lookup(users))
	server.SetChannelBinding([]byte("SERVER-sees-channel-B")) // different channel!

	err := runSCRAM(t, "SCRAM-SHA-256-PLUS", client, server)
	if err == nil {
		t.Fatal("expected channel-binding mismatch to fail, got success")
	}
	if !errors.Is(err, ErrChannelBinding) && !errors.Is(err, ErrAuthFailed) {
		t.Errorf("expected ErrChannelBinding/ErrAuthFailed, got %v", err)
	}
}

// TestSCRAMPlusFlagMismatch ensures a non-PLUS server rejects a client that
// tries to smuggle a "p=" channel-binding flag, and vice versa.
func TestSCRAMPlusFlagMismatch(t *testing.T) {
	users := map[string]string{"alice": "s3cret"}

	// PLUS client against a non-PLUS server.
	client := NewSCRAMSHA256Plus(Credentials{Username: "alice", Password: "s3cret", CBType: "tls-server-end-point", ChannelBinding: []byte("x")})
	server := NewSCRAMServerSHA256(lookup(users)) // non-PLUS
	if err := runSCRAM(t, "mismatch", client, server); !errors.Is(err, ErrChannelBinding) {
		t.Errorf("non-PLUS server should reject p= flag, got %v", err)
	}
}

func TestSCRAMWrongPassword(t *testing.T) {
	users := map[string]string{"alice": "s3cret"}
	client := NewSCRAMSHA256(Credentials{Username: "alice", Password: "WRONG"})
	server := NewSCRAMServerSHA256(lookup(users))

	err := runSCRAM(t, "SCRAM-SHA-256", client, server)
	if !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed for wrong password, got %v", err)
	}
	if !server.Completed() {
		t.Error("server should be marked completed even on failure")
	}
}

func TestSCRAMUnknownUser(t *testing.T) {
	client := NewSCRAMSHA256(Credentials{Username: "ghost", Password: "x"})
	server := NewSCRAMServerSHA256(lookup(map[string]string{"alice": "s3cret"}))

	// Start must NOT reveal the user is unknown (returns a normal server-first).
	clientFirst, err := client.Start()
	if err != nil {
		t.Fatal(err)
	}
	serverFirst, err := server.Start(clientFirst)
	if err != nil {
		t.Fatalf("server.Start on unknown user should not error, got %v", err)
	}
	if len(serverFirst) == 0 {
		t.Fatal("server-first should be non-empty for unknown user (anti-enumeration)")
	}
	clientFinal, err := client.Next(serverFirst)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.Finish(clientFinal); !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed for unknown user, got %v", err)
	}
}

func TestSCRAMServerMalformedInput(t *testing.T) {
	server := NewSCRAMServerSHA256(lookup(map[string]string{"alice": "s3cret"}))
	// Must not panic; must return an error.
	for _, in := range []string{"", "garbage", "n,", "n,,", "n,,x=1"} {
		if _, err := server.Start([]byte(in)); err == nil {
			t.Errorf("Start(%q) should return an error", in)
		}
		server = NewSCRAMServerSHA256(lookup(map[string]string{"alice": "s3cret"}))
	}
}

func TestNewSCRAMServerUnknownMechanism(t *testing.T) {
	if NewSCRAMServer("SCRAM-SHA-999", nil) != nil {
		t.Error("unknown mechanism should return nil")
	}
	if NewSCRAMServer("scram-sha-256", nil) == nil {
		t.Error("case-insensitive name should be accepted")
	}
}

func TestUnescapeSCRAM(t *testing.T) {
	// Username with , and = must round-trip through escape/unescape.
	orig := "a,b=c"
	escaped := escapeSCRAM(orig)
	if got := unescapeSCRAM(escaped); got != orig {
		t.Errorf("unescapeSCRAM(escapeSCRAM(%q)) = %q", orig, got)
	}
}
