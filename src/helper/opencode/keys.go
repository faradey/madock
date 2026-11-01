package opencode

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// The keys are madock's, generated once per project and kept under the
// runtime directory where nothing regenerates them.
//
// The host key in particular: openssh-server generates one when the package
// is installed, which is at image build time, so every rebuild would present
// a new host and the client would refuse it — or would have to be told not to
// check, which is the one setting this design cannot afford. With madock's
// key the known_hosts line is written here and checked strictly.
const (
	keysDir            = "opencode-keys"
	clientKeyFile      = "id_ed25519"
	hostKeyFile        = "hostkey"
	authorizedKeysFile = "authorized_keys"
	knownHostsFile     = "known_hosts"
)

// KeysDir is where EnsureKeys writes, under a project's runtime directory.
// Its content differs on every machine, so generated-file comparisons skip it.
const KeysDir = keysDir

// EnsureKeys generates the client and host keys if they are missing, and
// writes authorized_keys and known_hosts from them either way.
func EnsureKeys(runtimeDir string) error {
	dir := filepath.Join(runtimeDir, keysDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	client, err := ensureKey(filepath.Join(dir, clientKeyFile), "madock opencode client")
	if err != nil {
		return err
	}
	host, err := ensureKey(filepath.Join(dir, hostKeyFile), "madock opencode-exec host key")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, authorizedKeysFile), ssh.MarshalAuthorizedKey(client), 0o600); err != nil {
		return err
	}
	known := ExecService + " " + string(ssh.MarshalAuthorizedKey(host))
	return os.WriteFile(filepath.Join(dir, knownHostsFile), []byte(known), 0o600)
}

// ensureKey reads an ed25519 private key, generating it first if the file
// does not exist, and returns the public half. 0600 is not tidiness: ssh and
// sshd both refuse a private key others can read.
func ensureKey(path, comment string) (ssh.PublicKey, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		_, priv, genErr := ed25519.GenerateKey(rand.Reader)
		if genErr != nil {
			return nil, fmt.Errorf("generating %s: %w", path, genErr)
		}
		block, marshalErr := ssh.MarshalPrivateKey(priv, comment)
		if marshalErr != nil {
			return nil, fmt.Errorf("encoding %s: %w", path, marshalErr)
		}
		raw = pem.EncodeToMemory(block)
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	key, err := ssh.ParseRawPrivateKey(raw)
	if err != nil {
		return nil, fmt.Errorf("%s does not parse: %w", path, err)
	}
	// The OpenSSH format parses to a pointer, PEM/PKCS8 to a value; both are
	// the same key.
	switch priv := key.(type) {
	case ed25519.PrivateKey:
		return ssh.NewPublicKey(priv.Public())
	case *ed25519.PrivateKey:
		return ssh.NewPublicKey(priv.Public())
	}
	return nil, fmt.Errorf("%s is not an ed25519 key", path)
}
