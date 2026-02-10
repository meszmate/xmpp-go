//go:build integration

package mysql_test

import (
	"os"
	"testing"

	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/mysql"
	"github.com/meszmate/xmpp-go/storage/storagetest"
)

func TestMySQLStorage(t *testing.T) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		t.Skip("MYSQL_DSN not set; skipping integration test")
	}

	storagetest.TestStorage(t, func() storage.Storage {
		s, err := mysql.New(dsn)
		if err != nil {
			t.Fatal(err)
		}
		return s
	})
}
