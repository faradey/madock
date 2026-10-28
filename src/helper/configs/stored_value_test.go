package configs

import (
	"os"
	"path/filepath"
	"testing"
)

func setUpStoredValueProject(t *testing.T, runtimeConfig string) string {
	t.Helper()
	exec := t.TempDir()
	run := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", exec)
	t.Setenv("MADOCK_RUN_DIR", run)
	CleanCache()
	t.Cleanup(CleanCache)

	project := filepath.Base(run)
	registry := filepath.Join(exec, "projects", project)
	if err := os.MkdirAll(registry, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(registry, "config.xml"), []byte(runtimeConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	return project
}

// The bug this exists for: a key removed from every file still reads back from
// the merged configuration, at its default. Reporting that as "still set" named
// a file that had nothing in it.
func TestStoredValue_ADefaultIsNotAStoredValue(t *testing.T) {
	project := setUpStoredValueProject(t, `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <platform>custom</platform>
        </default>
    </scopes>
</config>
`)

	if _, ok := GetCurrentProjectConfig()["varnish/enabled"]; !ok {
		t.Fatal("precondition: the merged configuration should carry the default for varnish/enabled")
	}
	if value, file, ok := StoredValue(project, "varnish/enabled"); ok {
		t.Errorf("a default was reported as stored: %q in %s", value, file)
	}
}

func TestStoredValue_FindsTheMachineCopy(t *testing.T) {
	project := setUpStoredValueProject(t, `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <varnish>
                <enabled>true</enabled>
            </varnish>
        </default>
    </scopes>
</config>
`)

	value, file, ok := StoredValue(project, "varnish/enabled")
	if !ok {
		t.Fatal("a value stored in the machine's copy was not found")
	}
	if value != "true" {
		t.Errorf("value = %q, want %q", value, "true")
	}
	if filepath.Base(filepath.Dir(file)) != project {
		t.Errorf("the value was attributed to %s", file)
	}
}
