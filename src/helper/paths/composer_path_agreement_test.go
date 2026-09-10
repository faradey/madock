package paths_test

import (
	"os"
	"testing"

	"github.com/faradey/madock/v4/src/helper/paths"
	"github.com/faradey/madock/v4/src/model/versions/magento2"
)

// The production code in `model/versions/magento2` reads MADOCK_RUN_DIR itself
// rather than asking `helper/paths`, because that import is what made a model
// depend on a helper that depends on models. This test keeps the two answers in
// step.
//
// It lives here rather than beside the code it checks, and the architecture
// check is why: a model may not import a helper, and a test file counts. Here
// the direction is helper → model, which is the way the layers run.
//
// If `paths` ever learns another way to find the run directory, this goes red
// rather than the version detection quietly reading the wrong composer.json.
func TestTheComposerPathAgreesWithTheRunDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_RUN_DIR", dir)

	if got, want := magento2.ComposerJSON(), paths.GetRunDirPath()+"/composer.json"; got != want {
		t.Errorf("composer.json read from %q, while the run directory says %q", got, want)
	}
}

// And with no environment variable at all, both fall back to the working
// directory.
func TestWithoutTheEnvironmentBothFallBackToTheWorkingDirectory(t *testing.T) {
	t.Setenv("MADOCK_RUN_DIR", "")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if got := magento2.ComposerJSON(); got != wd+"/composer.json" {
		t.Errorf("composer.json read from %q, expected %q", got, wd+"/composer.json")
	}
}
