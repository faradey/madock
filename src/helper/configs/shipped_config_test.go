package configs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestShippedConfigMatchesEmbedded keeps madock's two files of defaults from
// disagreeing, because when they disagree the winner is the one nobody expects.
//
// There are two on purpose: config_defaults.xml is compiled into the binary and
// is the only source an installation from a downloaded release has, while the
// config.xml at the root is what install.sh leaves on disk — and
// GetOriginalGeneralConfig reads that one **over** the embedded defaults. So the
// file that is easiest to forget is the one that wins.
//
// It had been forgotten. Measured on 2026-08-15, the root copy was a strict
// subset missing seventy-three keys — every one of php/sendmail, proxy/mailpit,
// memcached, artemis, search/meilisearch, permissions/umask and the storefront
// blocks — and it disagreed on a value: an installation from source ran
// Europe/Kiev while one from a release ran UTC. Nobody chose that; it is what a
// stale copy with the last word produces.
//
// Comparing bytes rather than parsed keys is deliberate. The root copy is what
// docs/customizations.md points a user at to read the available settings, so its
// comments have to be the current ones too.
func TestShippedConfigMatchesEmbedded(t *testing.T) {
	root := repoRoot(t)

	shipped, err := os.ReadFile(filepath.Join(root, "config.xml"))
	if err != nil {
		t.Fatalf("reading the shipped config.xml: %v", err)
	}

	if string(shipped) != string(defaultConfigXML) {
		t.Errorf("config.xml at the repository root differs from src/helper/configs/config_defaults.xml.\n"+
			"The root copy is read over the embedded defaults, so a difference decides what an\n"+
			"installation from source does and an installation from a release does not.\n"+
			"Copy it: cp src/helper/configs/config_defaults.xml config.xml\n"+
			"(shipped %d bytes, embedded %d)", len(shipped), len(defaultConfigXML))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate this test file")
	}

	// src/helper/configs → repository root
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
}
