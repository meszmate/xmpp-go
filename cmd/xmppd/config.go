package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Domain           string
	Addr             string
	TLSCert          string
	TLSKey           string
	TLSSelfSigned    bool
	TLSSelfSignedDir string
	Storage          string
	StorageDSN       string
	StoragePath      string
	MongoDBName      string
	Plugins          []string
	DefaultAccounts  []Account
	CapsNode         string
	VersionName      string
	VersionString    string
	OMEMODeviceID    uint32
	Registration     registrationConfig
}

type Account struct {
	Username string
	Password string
}

func loadConfig() Config {
	cfg := Config{}
	cfg.Domain = getenv("XMPP_DOMAIN", "example.com")
	cfg.Addr = getenv("XMPP_ADDR", ":5222")
	cfg.TLSCert = os.Getenv("XMPP_TLS_CERT")
	cfg.TLSKey = os.Getenv("XMPP_TLS_KEY")
	cfg.TLSSelfSigned = getenvBool("XMPP_TLS_SELF_SIGNED", false)
	cfg.TLSSelfSignedDir = getenv("XMPP_TLS_SELF_SIGNED_DIR", "/var/lib/xmpp/tls")
	cfg.Storage = strings.ToLower(getenv("XMPP_STORAGE", "file"))
	cfg.StorageDSN = os.Getenv("XMPP_STORAGE_DSN")
	cfg.StoragePath = getenv("XMPP_STORAGE_PATH", "/var/lib/xmpp/data")
	cfg.MongoDBName = getenv("XMPP_MONGO_DB", "xmpp")
	cfg.Plugins = parseCSV(getenv("XMPP_PLUGINS", "disco,roster,presence,ping,vcard,time,version"))
	cfg.DefaultAccounts = parseAccounts(os.Getenv("XMPP_DEFAULT_ACCOUNTS"))
	cfg.CapsNode = getenv("XMPP_CAPS_NODE", "xmpp-go")
	cfg.VersionName = getenv("XMPP_VERSION_NAME", "xmpp-go")
	cfg.VersionString = getenv("XMPP_VERSION", "dev")
	cfg.OMEMODeviceID = uint32(getenvInt("XMPP_OMEMO_DEVICE_ID", 1))
	cfg.Registration = registrationConfig{
		Policy:       registrationPolicy(strings.ToLower(getenv("XMPP_REGISTRATION_POLICY", "open"))),
		Fields:       parseCSV(getenv("XMPP_REGISTRATION_FIELDS", "username,password,email")),
		Invites:      parseTokenSet(os.Getenv("XMPP_REGISTRATION_INVITES")),
		AdminTokens:  parseTokenSet(os.Getenv("XMPP_REGISTRATION_ADMIN_TOKENS")),
		RateLimit:    getenvInt("XMPP_REGISTRATION_RATE_LIMIT", 5),
		RateWindow:   getenvDuration("XMPP_REGISTRATION_RATE_WINDOW", 1*time.Minute),
		Iterations:   getenvInt("XMPP_REGISTRATION_SCRAM_ITERATIONS", 4096),
		DataForm:     getenvBool("XMPP_REGISTRATION_DATAFORM", true),
		Instructions: getenv("XMPP_REGISTRATION_INSTRUCTIONS", "Fill out the form to create an account."),
	}
	return cfg
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func parseCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func parseAccounts(v string) []Account {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]Account, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, ":", 2)
		if len(kv) != 2 {
			continue
		}
		user := strings.TrimSpace(kv[0])
		pass := strings.TrimSpace(kv[1])
		if user == "" || pass == "" {
			continue
		}
		out = append(out, Account{Username: user, Password: pass})
	}
	return out
}

func parseTokenSet(v string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, p := range parseCSV(v) {
		out[p] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
