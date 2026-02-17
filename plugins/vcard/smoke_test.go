package vcard

import (
	"testing"

	"github.com/meszmate/xmpp-go/internal/testutil/pluginsmoke"
)

func TestPluginSmoke(t *testing.T) {
	pluginsmoke.Run(t, New())
}
