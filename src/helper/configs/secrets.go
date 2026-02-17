package configs

// SecretKeys contains config keys that hold sensitive data.
var SecretKeys = map[string]bool{
	"db/root_password":               true,
	"db/password":                    true,
	"ssh/password":                   true,
	"ssh/key_path":                   true,
	"magento/admin_password":         true,
	"magento/cloud/password":         true,
	"magento/mftf/otp_shared_secret": true,
}

// RegisterSecretKey marks an additional config key as secret.
func RegisterSecretKey(key string) {
	SecretKeys[key] = true
}

// SecretsProvider allows enterprise to encrypt/decrypt secret config values.
type SecretsProvider interface {
	Encrypt(key, plaintext string) (string, error)
	Decrypt(key, ciphertext string) (string, error)
}

var secretsProvider SecretsProvider

// SetSecretsProvider sets a custom secrets provider for encrypting/decrypting config values.
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
func decryptIfSecret(key, value string) string {
	if secretsProvider == nil || !isSecretKey(key) {
		return value
	}
	decrypted, err := secretsProvider.Decrypt(key, value)
	if err != nil {
		return value
	}
	return decrypted
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
