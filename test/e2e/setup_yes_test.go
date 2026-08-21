//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSetupYesRefusesAQuestionItCannotAnswer covers the questions `--yes` had no
// answer for.
//
// Every other question honours the flag: the selectors take the configured
// default, or the first real option. The platform-version prompts are asked only
// when nothing supplied a version — no preset, nothing detected in the project
// directory, no default in the configuration — and in that case there was
// nothing for `--yes` to fall back on. So it asked anyway, read EOF from a closed
// stdin, and ended the command with "logger.go:82: EOF", which names neither the
// question nor the flag that answers it. A scripted
// `madock setup -y --platform=shopware` died there every time.
//
// Shopware is the platform with no default version, which is why it is the one
// under test. Magento is not: it has defaults, so `-y` proceeds without
// --platform-version and this refusal never applies to it — a test written
// against Magento passes for the wrong reason, and then spends eleven minutes
// building images to do it.
func TestSetupYesRefusesAQuestionItCannotAnswer(t *testing.T) {
	p := newProject(t, "e2esetupyes")

	out, err := p.tryRun(3*time.Minute, "setup", "-y",
		"--platform=shopware",
		"--hosts=e2esetupyes.test",
	)
	if err == nil {
		t.Fatalf("setup -y with no --platform-version was accepted:\n%s", out)
	}

	// The flag by name. The refusal exists to end a search, so naming the
	// question without naming the answer would only move the search.
	requireContains(t, out, "--platform-version", "the refusal naming the flag that answers it")

	// Not the old failure. EOF describes the machinery — a closed stdin — rather
	// than the mistake, and its return is what this test watches for.
	if strings.Contains(out, "EOF") {
		t.Errorf("the command still died on a closed stdin rather than refusing:\n%s", out)
	}

	// And before doing anything, which is the half that matters when the command
	// is `setup -y -d -i`: the download and the install run in the same handler,
	// after the question. The generated directory is what setup writes first, so
	// its absence is the evidence that the refusal came before the work.
	generated := filepath.Join(p.execDir, "aruntime", "projects", p.name)
	if _, statErr := os.Stat(generated); statErr == nil {
		t.Errorf("the refusal left generated files behind at %s", generated)
	}
}
