package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/faradey/madock/v4/src/helper/configs"
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

// `config:set --global` writes projects/config.xml, so `config:unset --global`
// has to be able to take its settings back out. Without the flag the shared
// file is left alone.
func TestUnsetTargets_GlobalReachesTheSharedFile(t *testing.T) {
	exec := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", exec)

	shared := filepath.Join(exec, "projects", "config.xml")

	found := false
	for _, target := range unsetTargets("p", "dev", true) {
		if target.file == shared {
			found = true
			if target.scope != "default" {
				t.Errorf("the shared file is read under %q, not default", target.scope)
			}
		}
	}
	if !found {
		t.Error("--global does not reach projects/config.xml, which config:set --global writes")
	}

	for _, target := range unsetTargets("p", "dev", false) {
		if target.file == shared || target.file == filepath.Join(exec, "config.xml") {
			t.Errorf("without --global the installation's file %s is edited", target.file)
		}
	}
}

// A setting written with `config:set --global` is taken out by the same
// command's counterpart, and is no longer reported as stored.
func TestUnsetGlobal_RemovesASettingFromTheSharedFile(t *testing.T) {
	exec := t.TempDir()
	run := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", exec)
	t.Setenv("MADOCK_RUN_DIR", run)
	configs.CleanCache()
	t.Cleanup(configs.CleanCache)

	shared := filepath.Join(exec, "projects", "config.xml")
	if err := os.MkdirAll(filepath.Dir(shared), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shared, []byte(`<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <proxy>
                <worker>
                    <processes>3</processes>
                </worker>
            </proxy>
        </default>
    </scopes>
</config>
`), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, target := range unsetTargets(filepath.Base(run), "default", true) {
		if target.file != shared {
			continue
		}
		if err := configs.RemoveKeepingComments(target.file, []string{"proxy/worker/processes"}, target.scope); err != nil {
			t.Fatal(err)
		}
	}
	configs.CleanCache()

	if _, still := configs.ParseXmlFile(shared)["scopes/default/proxy/worker/processes"]; still {
		t.Error("the key still parses out of projects/config.xml")
	}
	if _, _, still := configs.StoredValue(filepath.Base(run), "proxy/worker/processes"); still {
		t.Error("the setting is still reported as stored after --global")
	}
}
