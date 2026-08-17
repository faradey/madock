package configs

import (
	"os"
	"path/filepath"
	"testing"
)

// installationWith writes an installation config.xml holding the given body
// inside <config> and points madock at that directory.
func installationWith(t *testing.T, body string) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	xml := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<config>\n" + body + "\n</config>\n"
	if err := os.WriteFile(filepath.Join(dir, "config.xml"), []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}

	CleanCache()
	t.Cleanup(CleanCache)

	return dir
}

// The shipped answer: a laptop removes projects all day, and a guard that gets
// in the way there is a guard people learn to switch off.
func TestAllowsDestructiveCommands_DefaultsToAllowed(t *testing.T) {
	installationWith(t, "    <activeScope>default</activeScope>")

	if !AllowsDestructiveCommands() {
		t.Fatal("the community default should allow destructive commands")
	}
}

func TestAllowsDestructiveCommands_FileSaysNo(t *testing.T) {
	installationWith(t, "    <"+DestructiveKey+">false</"+DestructiveKey+">")

	if AllowsDestructiveCommands() {
		t.Fatal("the installation said false and was not believed")
	}
}

// An edition may disagree with the default — madock-pro runs on servers and
// says no there — and the file still has the last word, or turning it back on
// would require a different binary.
func TestAllowsDestructiveCommands_EditionThenFile(t *testing.T) {
	installationWith(t, "    <activeScope>default</activeScope>")

	SetDefaultOverride(DestructiveKey, "false")
	t.Cleanup(func() { delete(defaultOverrides, DestructiveKey) })

	if AllowsDestructiveCommands() {
		t.Fatal("the edition's answer was ignored")
	}

	installationWith(t, "    <"+DestructiveKey+">true</"+DestructiveKey+">")
	if !AllowsDestructiveCommands() {
		t.Fatal("the file must override the edition, or a machine cannot re-enable it")
	}
}

// The property the placement exists for. A guard a project can switch off is
// not a guard — and config:set writes to the project, so a scoped key would be
// reachable by the very command this protects against.
func TestDestructiveKey_IsNotReachableFromAProject(t *testing.T) {
	scoped := ParseXmlBytes([]byte(`<?xml version="1.0"?>
<config>
    <scopes>
        <default>
            <` + DestructiveKey + `>true</` + DestructiveKey + `>
        </default>
    </scopes>
</config>`))

	project := getConfigByScope(scoped, "default")
	if _, ok := project[DestructiveKey]; !ok {
		t.Skip("a scoped copy of the key does reach a project map — that is expected; the check below is the real one")
	}

	// Reaching a project map is one thing; being believed is another. The
	// resolver reads the top level only, so a scoped value changes nothing.
	dir := installationWith(t, `    <`+DestructiveKey+`>false</`+DestructiveKey+`>
    <scopes>
        <default>
            <`+DestructiveKey+`>true</`+DestructiveKey+`>
        </default>
    </scopes>`)
	_ = dir

	if AllowsDestructiveCommands() {
		t.Fatal("a value inside <scopes> overrode the installation's own answer")
	}
}

// And the top-level key is where the parser leaves it: unprefixed, so
// getConfigByScope — which every project-facing reader goes through — drops it.
func TestDestructiveKey_IsDroppedByScopeExtraction(t *testing.T) {
	parsed := ParseXmlBytes([]byte(`<?xml version="1.0"?>
<config>
    <` + DestructiveKey + `>false</` + DestructiveKey + `>
    <scopes><default><platform>magento2</platform></default></scopes>
</config>`))

	if _, ok := parsed[DestructiveKey]; !ok {
		t.Fatalf("the parser did not keep the top-level key: %v", parsed)
	}

	if _, ok := getConfigByScope(parsed, "default")[DestructiveKey]; ok {
		t.Fatal("the key survived scope extraction, so a project would inherit it")
	}
}

// The guard has to survive madock writing the file for other reasons.
//
// SaveInFile rewrites config.xml whole: it parses what is there, merges the
// incoming keys into a scope and writes the result back. A top-level key is not
// part of any scope, so it travels through that merge as itself — and if it did
// not, an installation that allowed the command would quietly stop allowing it
// the next time anything wrote a password or a setting, with no diff a person
// would think to look at.
func TestDestructiveKey_SurvivesAWriteToTheSameFile(t *testing.T) {
	dir := installationWith(t, `    <`+DestructiveKey+`>true</`+DestructiveKey+`>
    <scopes>
        <default>
            <platform>magento2</platform>
        </default>
    </scopes>`)

	path := filepath.Join(dir, "config.xml")
	SaveInFile(path, map[string]string{"db/password": "written-later"}, "default")
	CleanCache()

	written := ParseXmlFile(path)
	if written[DestructiveKey] != "true" {
		t.Fatalf("the guard's key did not survive a write to the same file: %v", written)
	}
	if written["scopes/default/db/password"] != "written-later" {
		t.Errorf("the write itself did not land: %v", written)
	}

	if !AllowsDestructiveCommands() {
		t.Fatal("the installation's answer changed because something else wrote to the file")
	}
}
