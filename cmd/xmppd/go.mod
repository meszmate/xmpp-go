module github.com/meszmate/xmpp-go/cmd/xmppd

go 1.25.0

require (
	github.com/meszmate/xmpp-go v0.0.0
	github.com/meszmate/xmpp-go/storage/mongodb v0.0.0
	github.com/meszmate/xmpp-go/storage/mysql v0.0.0
	github.com/meszmate/xmpp-go/storage/postgres v0.0.0
	github.com/meszmate/xmpp-go/storage/redis v0.0.0
	github.com/meszmate/xmpp-go/storage/sqlite v0.0.0
	github.com/redis/go-redis/v9 v9.9.0
	golang.org/x/crypto v0.47.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-sql-driver/mysql v1.9.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.4 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/mattn/go-sqlite3 v1.14.28 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.mongodb.org/mongo-driver/v2 v2.2.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/text v0.33.0 // indirect
)

replace (
	github.com/meszmate/xmpp-go => ../..
	github.com/meszmate/xmpp-go/storage/mongodb => ../../storage/mongodb
	github.com/meszmate/xmpp-go/storage/mysql => ../../storage/mysql
	github.com/meszmate/xmpp-go/storage/postgres => ../../storage/postgres
	github.com/meszmate/xmpp-go/storage/redis => ../../storage/redis
	github.com/meszmate/xmpp-go/storage/sqlite => ../../storage/sqlite
)
