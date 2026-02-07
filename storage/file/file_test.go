package file_test

import (
	"testing"

	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/file"
	"github.com/meszmate/xmpp-go/storage/storagetest"
)

func TestFileStorage(t *testing.T) {
	storagetest.TestStorage(t, func() storage.Storage {
		return file.New(t.TempDir())
	})
}
