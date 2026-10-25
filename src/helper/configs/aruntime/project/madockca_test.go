package project

import (
	"os"
	"path/filepath"
	"testing"
)

// The installation's CA goes into every project's build context, where the
// application images install it; without a CA the file is there and empty,
// so the image's COPY never fails and the install step is skipped.
func TestTheInstallationsCAReachesTheBuildContext(t *testing.T) {
	old := madockCA
	t.Cleanup(func() { madockCA = old })
	ctx := t.TempDir()

	madockCA = func() string { return "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n" }
	writeCtxFiles("shop", ctx)
	got, err := os.ReadFile(filepath.Join(ctx, madockCAFile))
	if err != nil || string(got) != madockCA() {
		t.Fatalf("ctx/%s holds %q (%v), want the installation's CA", madockCAFile, got, err)
	}

	madockCA = func() string { return "" }
	writeCtxFiles("shop", ctx)
	if info, err := os.Stat(filepath.Join(ctx, madockCAFile)); err != nil || info.Size() != 0 {
		t.Errorf("without a CA ctx/%s should exist and be empty: %v %v", madockCAFile, info, err)
	}
}
