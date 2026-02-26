package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	xmpp "github.com/meszmate/xmpp-go"
	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/file"
	"github.com/meszmate/xmpp-go/storage/memory"
	"github.com/meszmate/xmpp-go/storage/mongodb"
	"github.com/meszmate/xmpp-go/storage/mysql"
	"github.com/meszmate/xmpp-go/storage/postgres"
	"github.com/meszmate/xmpp-go/storage/redis"
	"github.com/meszmate/xmpp-go/storage/sqlite"

	redislib "github.com/redis/go-redis/v9"
)

func main() {
	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.TLSSelfSigned && (cfg.TLSCert == "" || cfg.TLSKey == "") {
		certPath, keyPath, err := ensureSelfSigned(cfg)
		if err != nil {
			log.Fatalf("tls: %v", err)
		}
		cfg.TLSCert = certPath
		cfg.TLSKey = keyPath
	}
	if cfg.Domain == "example.com" {
		log.Printf("warning: XMPP_DOMAIN is set to example.com (default). Set it to your real domain.")
	}

	store, err := buildStorage(cfg)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	plugins, err := buildPlugins(cfg)
	if err != nil {
		log.Fatalf("plugins: %v", err)
	}

	var seedOnce sync.Once
	var seedErr error

	opts := []xmpp.ServerOption{
		xmpp.WithServerAddr(cfg.Addr),
	}
	if store != nil {
		opts = append(opts, xmpp.WithServerStorage(store))
	}
	if len(plugins) > 0 {
		opts = append(opts, xmpp.WithServerPlugins(plugins...))
	}
	opts = append(opts, xmpp.WithServerSessionHandler(func(ctx context.Context, session *xmpp.Session) {
		seedOnce.Do(func() {
			if store == nil {
				return
			}
			if err := seedDefaultAccounts(ctx, store, cfg.DefaultAccounts); err != nil {
				seedErr = err
			}
		})
		if seedErr != nil {
			log.Printf("seed accounts: %v", seedErr)
			_ = session.Close()
			return
		}
		serveSession(ctx, session, cfg, store)
	}))

	server, err := xmpp.NewServer(cfg.Domain, opts...)
	if err != nil {
		log.Fatalf("server: %v", err)
	}

	log.Printf("xmpp-go server starting domain=%s addr=%s storage=%s", cfg.Domain, cfg.Addr, cfg.Storage)
	if err := server.ListenAndServe(ctx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		log.Fatalf("server: %v", err)
	}
}

func buildStorage(cfg Config) (storage.Storage, error) {
	switch cfg.Storage {
	case "", "memory":
		return memory.New(), nil
	case "file":
		if err := os.MkdirAll(cfg.StoragePath, 0o755); err != nil {
			return nil, err
		}
		return file.New(cfg.StoragePath), nil
	case "sqlite":
		dsn := cfg.StorageDSN
		if dsn == "" {
			dsn = filepath.Join(cfg.StoragePath, "xmpp.db")
			if err := os.MkdirAll(cfg.StoragePath, 0o755); err != nil {
				return nil, err
			}
		}
		return sqlite.New(dsn)
	case "postgres":
		if cfg.StorageDSN == "" {
			return nil, fmt.Errorf("XMPP_STORAGE_DSN is required for postgres")
		}
		return postgres.New(cfg.StorageDSN)
	case "mysql":
		if cfg.StorageDSN == "" {
			return nil, fmt.Errorf("XMPP_STORAGE_DSN is required for mysql")
		}
		return mysql.New(cfg.StorageDSN)
	case "mongodb", "mongo":
		if cfg.StorageDSN == "" {
			return nil, fmt.Errorf("XMPP_STORAGE_DSN is required for mongodb")
		}
		return mongodb.New(cfg.StorageDSN, cfg.MongoDBName)
	case "redis":
		if cfg.StorageDSN == "" {
			return nil, fmt.Errorf("XMPP_STORAGE_DSN is required for redis")
		}
		opts, err := redislib.ParseURL(cfg.StorageDSN)
		if err != nil {
			return nil, err
		}
		return redis.New(opts), nil
	default:
		return nil, fmt.Errorf("unknown storage: %s", cfg.Storage)
	}
}

func seedDefaultAccounts(ctx context.Context, st storage.Storage, accounts []Account) error {
	if len(accounts) == 0 {
		return nil
	}
	us := st.UserStore()
	if us == nil {
		return fmt.Errorf("storage backend does not support users")
	}
	for _, acc := range accounts {
		exists, err := us.UserExists(ctx, acc.Username)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if err := us.CreateUser(ctx, &storage.User{Username: acc.Username, Password: acc.Password}); err != nil {
			return err
		}
	}
	return nil
}

func ensureSelfSigned(cfg Config) (string, string, error) {
	if err := os.MkdirAll(cfg.TLSSelfSignedDir, 0o700); err != nil {
		return "", "", err
	}
	certPath := filepath.Join(cfg.TLSSelfSignedDir, "cert.pem")
	keyPath := filepath.Join(cfg.TLSSelfSignedDir, "key.pem")
	if fileExists(certPath) && fileExists(keyPath) {
		return certPath, keyPath, nil
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", err
	}

	hosts := []string{cfg.Domain}
	if cfg.Addr != "" {
		if host, _, err := net.SplitHostPort(cfg.Addr); err == nil && host != "" {
			hosts = append(hosts, host)
		}
	}

	notBefore := time.Now().Add(-time.Hour)
	notAfter := time.Now().Add(365 * 24 * time.Hour)

	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: cfg.Domain},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{},
		IPAddresses:  []net.IP{},
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
			continue
		}
		tmpl.DNSNames = append(tmpl.DNSNames, h)
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return "", "", err
	}

	if err := writePEM(certPath, "CERTIFICATE", derBytes, 0o644); err != nil {
		return "", "", err
	}
	keyBytes := x509.MarshalPKCS1PrivateKey(priv)
	if err := writePEM(keyPath, "RSA PRIVATE KEY", keyBytes, 0o600); err != nil {
		return "", "", err
	}

	return certPath, keyPath, nil
}

func writePEM(path, typ string, der []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: typ, Bytes: der})
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
