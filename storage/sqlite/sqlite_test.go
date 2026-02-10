package sqlite_test

import (
	"testing"

	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/sqlite"
	"github.com/meszmate/xmpp-go/storage/storagetest"
)

func TestSQLiteStorage(t *testing.T) {
	storagetest.TestStorage(t, func() storage.Storage {
		s, err := sqlite.New(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		return s
	})
}
