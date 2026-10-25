//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAnUnmatchedPresetIsRefused — `setup -y --preset magento-247`, the example
// the help gave, matched no preset, and setup warned and went on with manual
// configuration; under -y that is the defaults, so a script asking for 2.4.7
// got 2.4.9 (found by the Indemorio engine, 2026-10-25). Refused now, before
// anything is written, naming the presets that exist.
func TestAnUnmatchedPresetIsRefused(t *testing.T) {
	p := newProject(t, "e2epreset")
	out, err := p.tryRun(setupTimeout, "setup", "-y", "--platform=magento2", "--preset=magento-247", "--hosts=e2epreset.test")
	if err == nil {
		t.Fatalf("setup with a preset that matches nothing was accepted:\n%s", out)
	}
	requireContains(t, out, "No magento2 preset matches --preset magento-247", "the refusal naming the preset")
	requireContains(t, out, "2.4.8", "the presets that do exist")
	if _, statErr := os.Stat(filepath.Join(p.execDir, "aruntime", "projects", p.name)); statErr == nil {
		t.Error("the refusal left generated files behind")
	}
	if _, statErr := os.Stat(filepath.Join(p.execDir, "projects", p.name)); statErr == nil {
		t.Error("the refusal left an empty projects/" + p.name + " behind, an entry for a project that never was")
	}
}
