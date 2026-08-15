package remote_sync

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	encodingpem "encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func pem(block *encodingpem.Block) []byte {
	return encodingpem.EncodeToMemory(block)
}

func TestDialAgentWithoutSocket(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")

	if conn := dialAgent(); conn != nil {
		conn.Close()
		t.Fatal("expected no agent when SSH_AUTH_SOCK is unset")
	}
}

// A socket path left behind by a dead agent must not be treated as an agent —
// the dial fails and the key file is the better answer.
func TestDialAgentWithStaleSocket(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", filepath.Join(t.TempDir(), "gone.sock"))

	if conn := dialAgent(); conn != nil {
		conn.Close()
		t.Fatal("expected no agent for a socket that cannot be dialled")
	}
}

func TestDialAgentWithListeningSocket(t *testing.T) {
	sock := listeningSocket(t)
	t.Setenv("SSH_AUTH_SOCK", sock)

	conn := dialAgent()
	if conn == nil {
		t.Fatal("expected an agent connection when the socket accepts them")
	}
	conn.Close()
}

// The defect this file exists for: the agent and the key file used to be two
// entries in config.Auth, and to the protocol both answer to the name
// "publickey" — which is what the client marks as tried. An agent that refused
// therefore closed the path to ssh/key_path, and the configured key was never
// offered at all.
//
// The socket only has to accept a connection for the old code to take its
// two-method branch, so no agent protocol is needed to pin this.
func TestPublicKeyIsOneMethodEvenWithAnAgent(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", listeningSocket(t))

	methods := authMethods("key", "", writeKey(t, "id_ed25519", ed25519Key(t)))
	if len(methods) != 1 {
		t.Fatalf("public-key auth was offered as %d methods; the protocol has one, and the second is never tried", len(methods))
	}
}

func TestPasswordAuthIsUnchanged(t *testing.T) {
	methods := authMethods("password", "hunter2", "")
	if len(methods) != 1 {
		t.Fatalf("password auth built %d methods, want 1", len(methods))
	}
}

// The public half has to be found without a passphrase, or the key file cannot
// be offered while an agent key is still being tried. Three places it can come
// from, and the third is the one that matters: an encrypted OpenSSH key carries
// its public key in the clear.
func TestPublicKeyOfFindsTheHalfThatNeedsNoPassphrase(t *testing.T) {
	key := ed25519Key(t)

	t.Run("from the .pub beside it", func(t *testing.T) {
		path := writeKey(t, "with_pub", key)
		if err := os.WriteFile(path+".pub", ssh.MarshalAuthorizedKey(publicOf(t, key)), 0600); err != nil {
			t.Fatal(err)
		}
		// The private half is unreadable, so anything found came from the .pub.
		if err := os.WriteFile(path, []byte("not a key at all\n"), 0600); err != nil {
			t.Fatal(err)
		}

		if got := publicKeyOf(path); got == nil {
			t.Fatal("no public key found beside the key file")
		}
	})

	t.Run("from an unencrypted key", func(t *testing.T) {
		if got := publicKeyOf(writeKey(t, "plain", key)); got == nil {
			t.Fatal("no public key found in an unencrypted key file")
		}
	})

	t.Run("from an encrypted key, without the passphrase", func(t *testing.T) {
		block, err := ssh.MarshalPrivateKeyWithPassphrase(key, "", []byte("s3cret"))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "encrypted")
		if err := os.WriteFile(path, pem(block), 0600); err != nil {
			t.Fatal(err)
		}

		got := publicKeyOf(path)
		if got == nil {
			t.Fatal("an encrypted OpenSSH key carries its public key in the clear; it was not read")
		}
		if got.Type() != ssh.KeyAlgoED25519 {
			t.Errorf("public key type %q, want %q", got.Type(), ssh.KeyAlgoED25519)
		}
	})
}

// A key whose public half cannot be established is not offered at all. Not a
// nicety: the handshake dereferences the public key before anything can check
// it, so a signer without one is a panic rather than a skipped key.
func TestFileSignerRefusesAKeyItCannotDescribe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "garbage")
	if err := os.WriteFile(path, []byte("this is not a key\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if signer := fileSigner(path); signer != nil {
		t.Fatal("a key with no readable public half was offered to the handshake")
	}

	if signer := fileSigner(filepath.Join(t.TempDir(), "absent")); signer != nil {
		t.Fatal("a key file that does not exist was offered to the handshake")
	}
}

// An RSA key file has to sign with rsa-sha2-256. A plain ssh.Signer is assumed
// by the handshake to manage only ssh-rsa — SHA-1, refused by every OpenSSH
// since 8.8 — so the deferred signer has to declare the algorithm itself, or
// one silent refusal is traded for another on the commonest key type.
func TestLazyFileSignerSignsWithRsaSha2(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	signer := fileSigner(writeKey(t, "id_rsa", key))
	if signer == nil {
		t.Fatal("an RSA key file was not offered at all")
	}

	algorithmSigner, ok := signer.(ssh.AlgorithmSigner)
	if !ok {
		t.Fatal("the key file signer cannot be asked for an algorithm, so the handshake will offer ssh-rsa and be refused")
	}

	signature, err := algorithmSigner.SignWithAlgorithm(rand.Reader, []byte("payload"), ssh.KeyAlgoRSASHA256)
	if err != nil {
		t.Fatalf("signing with %s: %v", ssh.KeyAlgoRSASHA256, err)
	}
	if signature.Format != ssh.KeyAlgoRSASHA256 {
		t.Errorf("signature format %q, want %q", signature.Format, ssh.KeyAlgoRSASHA256)
	}
	if err := signer.PublicKey().Verify([]byte("payload"), signature); err != nil {
		t.Errorf("the signature does not verify against the key's own public half: %v", err)
	}
}

func ed25519Key(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func publicOf(t *testing.T, key any) ssh.PublicKey {
	t.Helper()
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return signer.PublicKey()
}

func writeKey(t *testing.T, name string, key any) string {
	t.Helper()
	block, err := ssh.MarshalPrivateKey(key, "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, pem(block), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// listeningSocket is a unix socket that accepts connections and speaks nothing.
// Enough for dialAgent, which only asks whether an agent is there.
//
// Not t.TempDir(): a unix socket path is capped at ~104 bytes and the per-test
// temp directory alone overruns it on macOS.
func listeningSocket(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("/tmp", "ma")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	sock := filepath.Join(dir, "a.sock")
	listener, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("cannot create unix socket: %v", err)
	}
	t.Cleanup(func() { listener.Close() })

	return sock
}
