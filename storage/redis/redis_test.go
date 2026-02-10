//go:build integration

package redis_test

import (
	"os"
	"testing"

	goredis "github.com/redis/go-redis/v9"

	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/redis"
	"github.com/meszmate/xmpp-go/storage/storagetest"
)

func TestRedisStorage(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR not set; skipping integration test")
	}

	storagetest.TestStorage(t, func() storage.Storage {
		return redis.New(&goredis.Options{
			Addr: addr,
		})
	})
}
