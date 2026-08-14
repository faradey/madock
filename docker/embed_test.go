package dockerassets

import (
	"io/fs"
	"os"
	"testing"
)

// TestEveryPlatformDirIsEmbedded guards the one failure mode of the go:embed
// list above: it is written by hand, and a platform added to this directory
// without a matching entry keeps working for everyone who runs from source
// (the templates are read off disk) while being absent from every built
// binary. bigcommerce, spree and sylius shipped that way — a `madock start`
// on those platforms died with "docker config file not found:
// docker-compose.yml" because nothing extracted them.
func TestEveryPlatformDirIsEmbedded(t *testing.T) {
	onDisk, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read docker dir: %v", err)
	}

	embedded := make(map[string]bool)
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatalf("read embedded FS: %v", err)
	}
	for _, e := range entries {
		embedded[e.Name()] = true
	}

	for _, e := range onDisk {
		if !e.IsDir() {
			continue
		}
		if !embedded[e.Name()] {
			t.Errorf("docker/%s exists on disk but is missing from the go:embed list in embed.go — it will not ship in a built binary", e.Name())
		}
	}
}
