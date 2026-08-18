package configs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const nodeConfig = `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <!-- node is pinned because the build scripts need 22 -->
            <nodejs>
                <version>22.22.0</version>
                <major_version>20</major_version>
            </nodejs>
            <php>
                <version>8.3</version>
            </php>
        </default>
    </scopes>
</config>
`

func writeConfigFile(t *testing.T, body string) string {
	t.Helper()

	file := filepath.Join(t.TempDir(), "config.xml")
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return file
}

// The stale copy goes, and nothing else does. The comment matters as much as the
// values: this shape is also a project's own committed config.xml.
func TestRemoveStoredDerivedTakesOnlyTheComputedKey(t *testing.T) {
	file := writeConfigFile(t, nodeConfig)

	removed, err := RemoveStoredDerived(file, "nodejs/version", "default")
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(removed) != 1 || removed[0] != "nodejs/major_version" {
		t.Fatalf("removed %v, want [nodejs/major_version]", removed)
	}

	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	body := string(after)

	if strings.Contains(body, "major_version") {
		t.Errorf("the stored derived key survived:\n%s", body)
	}
	for _, kept := range []string{"22.22.0", "<version>8.3</version>", "build scripts need 22"} {
		if !strings.Contains(body, kept) {
			t.Errorf("removal took %q with it:\n%s", kept, body)
		}
	}
}

// Nothing stored, nothing to do — and the file must not be rewritten for it.
func TestRemoveStoredDerivedLeavesAFileWithoutOneAlone(t *testing.T) {
	file := writeConfigFile(t, strings.Replace(nodeConfig, "                <major_version>20</major_version>\n", "", 1))
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	removed, err := RemoveStoredDerived(file, "nodejs/version", "default")
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(removed) != 0 {
		t.Fatalf("removed %v from a file that stores none", removed)
	}

	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("the file was rewritten for nothing:\n%s", string(after))
	}
}

// A key that governs nothing takes nothing with it, which is every other setting
// in the file.
func TestRemoveStoredDerivedIgnoresAnUnrelatedSource(t *testing.T) {
	file := writeConfigFile(t, nodeConfig)

	removed, err := RemoveStoredDerived(file, "php/version", "default")
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(removed) != 0 {
		t.Fatalf("php/version removed %v", removed)
	}
}

func TestDerivedFrom(t *testing.T) {
	if got := DerivedFrom("nodejs/version"); len(got) != 1 || got[0] != "nodejs/major_version" {
		t.Errorf("DerivedFrom(nodejs/version) = %v", got)
	}
	if got := DerivedFrom("nodejs/major_version"); len(got) != 0 {
		t.Errorf("the derived key governs %v — it is the result, not the source", got)
	}
}
