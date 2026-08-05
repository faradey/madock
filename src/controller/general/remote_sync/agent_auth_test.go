package remote_sync

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestAgentAuthWithoutSocket(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")

	if _, ok := AgentAuth(); ok {
		t.Fatal("expected no agent auth when SSH_AUTH_SOCK is unset")
	}
}

// A socket path left behind by a dead agent must not be treated as an agent —
// the dial fails and the caller has to fall back to the key file.
func TestAgentAuthWithStaleSocket(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", filepath.Join(t.TempDir(), "gone.sock"))

	if _, ok := AgentAuth(); ok {
		t.Fatal("expected no agent auth for a socket that cannot be dialled")
	}
}

func TestAgentAuthWithListeningSocket(t *testing.T) {
	// Not t.TempDir(): a unix socket path is capped at ~104 bytes and the
	// per-test temp directory alone overruns it on macOS.
	dir, err := os.MkdirTemp("/tmp", "ma")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	sock := filepath.Join(dir, "a.sock")

	listener, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("cannot create unix socket: %v", err)
	}
	defer listener.Close()

	t.Setenv("SSH_AUTH_SOCK", sock)

	method, ok := AgentAuth()
	if !ok {
		t.Fatal("expected agent auth when the socket accepts connections")
	}
	if method == nil {
		t.Fatal("expected a non-nil auth method")
	}
}
