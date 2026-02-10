//go:build integration

package mongodb_test

import (
	"os"
	"testing"

	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/mongodb"
	"github.com/meszmate/xmpp-go/storage/storagetest"
)

func TestMongoDBStorage(t *testing.T) {
	uri := os.Getenv("MONGO_URI")
	db := os.Getenv("MONGO_DB")
	if uri == "" || db == "" {
		t.Skip("MONGO_URI or MONGO_DB not set; skipping integration test")
	}

	storagetest.TestStorage(t, func() storage.Storage {
		s, err := mongodb.New(uri, db)
		if err != nil {
			t.Fatal(err)
		}
		return s
	})
}
