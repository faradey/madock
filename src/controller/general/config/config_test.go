package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/faradey/madock/v3/src/helper/configs"
)

// The gap config:unset closes, stated as the thing that actually happened: a
// setting deleted from the project's config in git went on applying on every
// machine that had ever run setup, because the machine-side copy still carried
// it and nothing could take it out.
func TestUnset_RemovesFromTheMachineCopy(t *testing.T) {
	exec := t.TempDir()
	run := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", exec)
	t.Setenv("MADOCK_RUN_DIR", run)
	configs.CleanCache()
	t.Cleanup(configs.CleanCache)

	project := filepath.Base(run)
	registry := filepath.Join(exec, "projects", project)
	if err := os.MkdirAll(registry, 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(registry, "config.xml")
	if err := os.WriteFile(path, []byte(`<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <platform>custom</platform>
            <custom_commands>
                <pr>
                    <command>git pull --rebase</command>
                </pr>
            </custom_commands>
        </default>
    </scopes>
</config>
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := configs.RemoveKeepingComments(path, []string{"custom_commands"}, "default"); err != nil {
		t.Fatal(err)
	}
	configs.CleanCache()

	parsed := configs.ParseXmlFile(path)
	for key := range parsed {
		if key == "scopes/default/custom_commands/pr/command" {
			t.Error("the retired command is still registered")
		}
	}
	if got := parsed["scopes/default/platform"]; got != "custom" {
		t.Errorf("an unrelated setting was taken with it: platform = %q", got)
	}
}
