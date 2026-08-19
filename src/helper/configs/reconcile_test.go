package configs

import (
	"os"
	"path/filepath"
	"testing"
)

// The measured case: a setting deleted from the project's config, committed and
// rolled out, went on living in every installation that had ever run setup.
//
// Pricesmith, 2026-08-17 — a `custom_commands` block was removed from the
// repository and `madock pr` kept working on every machine, with nothing failing
// and nobody told.
func TestAKeyTheProjectDeletedIsRemoved(t *testing.T) {
	snapshot := map[string]string{
		"scopes/default/custom_commands/pr": "git pull --rebase",
		"scopes/default/php/version":        "8.3",
	}
	current := map[string]string{
		"scopes/default/php/version": "8.3",
	}
	runtime := map[string]string{
		"scopes/default/custom_commands/pr": "git pull --rebase",
		"scopes/default/php/version":        "8.3",
		"scopes/default/path":               "/home/user/project",
	}

	result := compareForRemoval(snapshot, current, runtime)

	if len(result.Removed) != 1 || result.Removed[0] != "scopes/default/custom_commands/pr" {
		t.Fatalf("Removed = %v, want the deleted custom command", result.Removed)
	}
	if len(result.Kept) != 0 {
		t.Errorf("Kept = %v, want none", result.Kept)
	}
}

// A value changed on this machine is not the project's to remove.
//
// This is the distinction that did not exist before, and the reason the whole
// mechanism is a snapshot rather than a flag: two keys look identical in the
// installed file whether they were copied from the project or typed here with
// `config:set`. What tells them apart is whether the installed value is still
// the one the snapshot recorded.
func TestALocalChangeSurvivesTheProjectDeletingTheKey(t *testing.T) {
	snapshot := map[string]string{"scopes/default/php/version": "8.3"}
	current := map[string]string{}
	runtime := map[string]string{"scopes/default/php/version": "8.4"}

	result := compareForRemoval(snapshot, current, runtime)

	if len(result.Removed) != 0 {
		t.Errorf("Removed = %v; a value set on this machine is not the project's to drop", result.Removed)
	}
	if len(result.Kept) != 1 || result.Kept[0] != "scopes/default/php/version" {
		t.Fatalf("Kept = %v, want the locally changed key reported", result.Kept)
	}
}

// Only what the snapshot recorded is ever a candidate.
//
// Everything madock writes into the installed copy itself — `path`, generated
// passwords, ports — was never in the project's config, so it cannot be matched
// by a deletion there. That is the property that makes this safe to run on every
// rebuild rather than a thing to be careful with.
func TestKeysThatNeverCameFromTheProjectAreNotCandidates(t *testing.T) {
	snapshot := map[string]string{"scopes/default/php/version": "8.3"}
	current := map[string]string{"scopes/default/php/version": "8.3"}
	runtime := map[string]string{
		"scopes/default/php/version": "8.3",
		"scopes/default/path":        "/home/user/project",
		"scopes/default/db/password": "generated-here",
		"scopes/default/nginx/port":  "17003",
	}

	result := compareForRemoval(snapshot, current, runtime)

	if len(result.Removed) != 0 || len(result.Kept) != 0 {
		t.Fatalf("Removed = %v, Kept = %v; nothing was deleted from the project", result.Removed, result.Kept)
	}
}

// A key the project deleted that this installation never had is not news.
func TestAKeyAlreadyGoneIsNotReported(t *testing.T) {
	snapshot := map[string]string{"scopes/default/redis/enabled": "true"}
	current := map[string]string{}
	runtime := map[string]string{}

	result := compareForRemoval(snapshot, current, runtime)

	if len(result.Removed) != 0 || len(result.Kept) != 0 {
		t.Fatalf("Removed = %v, Kept = %v, want silence", result.Removed, result.Kept)
	}
}

// Structure is not a setting. `activeScope` sits outside every scope and says
// which one is in use — a project removing it is not asking for anything to be
// deleted from a machine.
func TestUnscopedEntriesAreLeftAlone(t *testing.T) {
	snapshot := map[string]string{"activeScope": "default"}
	current := map[string]string{}
	runtime := map[string]string{"activeScope": "default"}

	result := compareForRemoval(snapshot, current, runtime)

	if len(result.Removed) != 0 {
		t.Fatalf("Removed = %v; activeScope is structure, not a value", result.Removed)
	}
}

// Scopes are separate namespaces, and a deletion in one is not a deletion in
// another. A project that stops setting something for `production` while keeping
// it for `default` means exactly that.
func TestScopesAreIndependent(t *testing.T) {
	snapshot := map[string]string{
		"scopes/default/db/enabled":    "true",
		"scopes/production/db/enabled": "true",
	}
	current := map[string]string{
		"scopes/default/db/enabled": "true",
	}
	runtime := map[string]string{
		"scopes/default/db/enabled":    "true",
		"scopes/production/db/enabled": "true",
	}

	result := compareForRemoval(snapshot, current, runtime)

	if len(result.Removed) != 1 || result.Removed[0] != "scopes/production/db/enabled" {
		t.Fatalf("Removed = %v, want only the production key", result.Removed)
	}
}

// The end to end shape: two files on disk, a snapshot between them, and the
// installed config left without the key the project dropped — comments and all
// the rest of the document untouched, because the removal edits the text.
func TestReconcileRemovesFromTheInstalledFile(t *testing.T) {
	execDir := t.TempDir()
	runDir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", execDir)
	t.Setenv("MADOCK_RUN_DIR", runDir)
	CleanCache()

	projectName := "reconcile"
	projectDir := filepath.Join(execDir, "projects", projectName)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(runDir, ".madock"), 0o755); err != nil {
		t.Fatal(err)
	}

	withCommand := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <activeScope>default</activeScope>
    <scopes>
        <default>
            <!-- why this alias exists, which is what a comment is for -->
            <custom_commands>
                <pr>git pull --rebase</pr>
            </custom_commands>
            <php>
                <version>8.3</version>
            </php>
        </default>
    </scopes>
</config>`

	withoutCommand := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <activeScope>default</activeScope>
    <scopes>
        <default>
            <php>
                <version>8.3</version>
            </php>
        </default>
    </scopes>
</config>`

	runtimeFile := filepath.Join(projectDir, "config.xml")
	projectFile := filepath.Join(runDir, ".madock", "config.xml")

	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// setup: both copies hold the alias, and `path` points at the project.
	write(runtimeFile, withCommand)
	write(projectFile, withCommand)
	SetParam(projectName, "path", runDir, "default", MadockLevelConfigCode)
	CleanCache()

	// First run records the baseline and removes nothing — there is nothing yet
	// to compare against, and guessing would be worse than waiting.
	first, err := ReconcileRemovedProjectKeys(projectName)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Baseline {
		t.Fatalf("first run should record a baseline, got %+v", first)
	}
	if got := ParseXmlFile(runtimeFile)["scopes/default/custom_commands/pr"]; got == "" {
		t.Fatal("the baseline run removed something")
	}

	// The project drops the alias, commits, rolls out.
	write(projectFile, withoutCommand)

	second, err := ReconcileRemovedProjectKeys(projectName)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Removed) != 1 || second.Removed[0] != "scopes/default/custom_commands/pr" {
		t.Fatalf("Removed = %v, want the alias", second.Removed)
	}

	after := ParseXmlFile(runtimeFile)
	if _, still := after["scopes/default/custom_commands/pr"]; still {
		t.Error("the alias is still in the installed configuration")
	}
	if got := after["scopes/default/php/version"]; got != "8.3" {
		t.Errorf("php/version = %q; the rest of the file must be untouched", got)
	}

	// And it does not report the same removal twice: the snapshot moved with it.
	third, err := ReconcileRemovedProjectKeys(projectName)
	if err != nil {
		t.Fatal(err)
	}
	if len(third.Removed) != 0 {
		t.Errorf("Removed = %v on a second pass; the snapshot did not move", third.Removed)
	}
}
