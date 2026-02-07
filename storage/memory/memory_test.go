package memory_test

import (
	"testing"

	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/memory"
	"github.com/meszmate/xmpp-go/storage/storagetest"
)

func TestMemoryStorage(t *testing.T) {
	storagetest.TestStorage(t, func() storage.Storage {
		return memory.New()
	})
}
