//go:build integration

package postgres_test

import (
	"os"
	"testing"

	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/postgres"
	"github.com/meszmate/xmpp-go/storage/storagetest"
)

func TestPostgresStorage(t *testing.T) {
	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		t.Skip("PG_DSN not set; skipping integration test")
	}

	storagetest.TestStorage(t, func() storage.Storage {
		s, err := postgres.New(dsn)
		if err != nil {
			t.Fatal(err)
		}
		return s
	})
}
