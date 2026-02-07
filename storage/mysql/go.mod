module github.com/meszmate/xmpp-go/storage/mysql

go 1.25.0

require (
	github.com/go-sql-driver/mysql v1.9.2
	github.com/meszmate/xmpp-go v0.0.0
)

require filippo.io/edwards25519 v1.1.0 // indirect

replace github.com/meszmate/xmpp-go => ../..
