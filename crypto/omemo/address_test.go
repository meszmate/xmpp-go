package omemo

import "testing"

func TestAddressString(t *testing.T) {
	addr := Address{JID: "alice@example.com", DeviceID: 12345}
	want := "alice@example.com:12345"
	if got := addr.String(); got != want {
		t.Errorf("Address.String() = %q, want %q", got, want)
	}
}
