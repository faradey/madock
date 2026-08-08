package docker

import (
	"os"
	"path/filepath"
	"testing"
)

// A machine that has never used ssh must still be able to start a project.
//
// This is not hypothetical: the end-to-end suite failed on every CI run with
// "mkdir .../ssh: file exists", because the project's ssh entry is a symlink to
// ~/.ssh, the runner has no ~/.ssh, and Docker cannot make a directory out of a
// dangling link. Any fresh server behaves the same way; it only went unnoticed
// because every developer machine has had ~/.ssh for years.
func TestPrepareHomeDirsCreatesWhatTheMountsNeed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := prepareHomeDirs()
	if err != nil {
		t.Fatalf("prepareHomeDirs: %v", err)
	}
	if got != home {
		t.Fatalf("returned %q, want %q", got, home)
	}

	for _, dir := range []string{".composer", ".ssh"} {
		info, statErr := os.Stat(filepath.Join(home, dir))
		if statErr != nil {
			t.Fatalf("%s: %v", dir, statErr)
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}

	// known_hosts is bind-mounted as a file. If it is missing, Docker creates a
	// directory with that name inside the container instead.
	info, err := os.Stat(filepath.Join(home, ".ssh", "known_hosts"))
	if err != nil {
		t.Fatalf("known_hosts: %v", err)
	}
	if info.IsDir() {
		t.Error("known_hosts is a directory, which is exactly what the mount must not become")
	}
}

// Running twice must not disturb what the first run made, and must never
// truncate a real known_hosts — that file is the user's, and losing it turns
// every future ssh into a host-key prompt.
func TestPrepareHomeDirsLeavesExistingFilesAlone(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	knownHosts := filepath.Join(sshDir, "known_hosts")
	if err := os.WriteFile(knownHosts, []byte("example.com ssh-ed25519 AAAA\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := prepareHomeDirs(); err != nil {
		t.Fatalf("prepareHomeDirs: %v", err)
	}

	content, err := os.ReadFile(knownHosts)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "example.com ssh-ed25519 AAAA\n" {
		t.Errorf("known_hosts was rewritten: %q", string(content))
	}
}
