package configs

import (
	"sync"

	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
)

// SecretKeys contains config keys that hold sensitive data.
var SecretKeys = map[string]bool{
	"db/root_password":                   true,
	"db/password":                        true,
	"db2/root_password":                  true,
	"db2/password":                       true,
	"ssh/password":                       true,
	"ssh/key_path":                       true,
	"magento/admin_password":             true,
	"magento/cloud/password":             true,
	"magento/mftf/otp_shared_secret":     true,
	"rabbitmq/password":                  true,
	"grafana/auth/password":              true,
	"redis/auth/password":                true,
	"valkey/auth/password":               true,
	"search/elasticsearch/auth/password": true,
	"search/opensearch/auth/password":    true,
}

// RegisterSecretKey marks an additional config key as secret.
//
// Extension point for madock-pro, and seven keys arrive through it. Community
// has no encryption, so it registers none — which is why this looks unreachable
// here and is the whole mechanism there.
func RegisterSecretKey(key string) {
	SecretKeys[key] = true
}

// SecretsProvider allows enterprise to encrypt/decrypt secret config values.
type SecretsProvider interface {
	Encrypt(key, plaintext string) (string, error)
	Decrypt(key, ciphertext string) (string, error)
}

var secretsProvider SecretsProvider

// SetSecretsProvider installs the provider that encrypts and decrypts config
// values.
//
// Extension point for madock-pro. Nothing in this repository encrypts anything:
// the keys registered above are plain text in community and enciphered in pro,
// and this is the seam between those two answers.
func SetSecretsProvider(p SecretsProvider) {
	secretsProvider = p
}

// encryptIfSecret encrypts the value if the key is a known secret and a provider is set.
func encryptIfSecret(key, value string) string {
	if secretsProvider == nil || !isSecretKey(key) {
		return value
	}
	encrypted, err := secretsProvider.Encrypt(key, value)
	if err != nil {
		return value
	}
	return encrypted
}

// decryptIfSecret decrypts the value if the key is a known secret and a provider is set.
//
// A failure here used to be silent, and the silence is the expensive part. The
// value that comes back is then the ciphertext itself — the literal `ENC:…`
// string — so the database is handed a password nobody typed and answers
// `ERROR 1045 (28000): Access denied`. That message names the database, which
// is the one system that is working: the fault is a master key this machine
// cannot read.
//
// Measured on 2026-09-08: a command run under `sudo` reads `$HOME=/root`,
// finds no key there, generates a new one, and every encrypted value in the
// configuration decrypts to nothing. The diagnosis went to MySQL and cost half
// a day.
//
// Reported once per run rather than per value: a project config holds several
// secrets and they all fail together, so one line says it and a dozen bury it.
func decryptIfSecret(key, value string) string {
	if secretsProvider == nil || !isSecretKey(key) {
		return value
	}
	decrypted, err := secretsProvider.Decrypt(key, value)
	if err != nil {
		reportUnreadableSecret(key, err)
		return value
	}
	return decrypted
}

// secretReportOnce keeps the warning to one line per process.
var secretReportOnce sync.Once

func reportUnreadableSecret(key string, err error) {
	secretReportOnce.Do(func() {
		fmtc.WarningLn("A secret in this project's configuration could not be decrypted: " + key)
		fmtc.WarningLn("  " + err.Error())
		fmtc.WarningLn("  The value is being used as it is stored, so anything reading it — a database, an")
		fmtc.WarningLn("  admin tool, a mail relay — will refuse it and blame its own credentials.")
		fmtc.WarningLn("  This machine's master key is what cannot read it. Under sudo the key is looked for")
		fmtc.WarningLn("  in /root rather than in your home, and a missing one is generated rather than found.")
	})
}

// isSecretKey checks if a config key holds sensitive data.
// Handles scoped keys like "scopes/default/db/password".
func isSecretKey(key string) bool {
	if SecretKeys[key] {
		return true
	}
	// Strip scope prefix: "scopes/<scope>/<key>" → "<key>"
	if parts := splitScopeKey(key); parts != "" {
		return SecretKeys[parts]
	}
	return false
}

// splitScopeKey strips "scopes/<scope>/" prefix and returns the bare key.
func splitScopeKey(key string) string {
	const prefix = "scopes/"
	if len(key) > len(prefix) && key[:len(prefix)] == prefix {
		// Find second "/" after "scopes/"
		rest := key[len(prefix):]
		idx := 0
		for idx < len(rest) && rest[idx] != '/' {
			idx++
		}
		if idx < len(rest)-1 {
			return rest[idx+1:]
		}
	}
	return ""
}
