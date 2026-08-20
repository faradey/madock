package versions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/configs"
)

// installation lays out a madock installation with one project in it and points
// madock at it, returning the paths the migration has to reach.
func installation(t *testing.T) (execDir, registry, projectDir string) {
	t.Helper()

	root := t.TempDir()
	execDir = filepath.Join(root, "install")
	projectDir = filepath.Join(root, "shop")

	registry = filepath.Join(execDir, "projects", "shop")
	for _, dir := range []string{registry, filepath.Join(projectDir, ".madock", "docker", "php")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("MADOCK_EXEC_DIR", execDir)
	configs.CleanCache()
	t.Cleanup(configs.CleanCache)

	return execDir, registry, projectDir
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

const oldSpelling = `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <platform>custom</platform>
            <language>php</language>
            <php>
                <version>8.2</version>
                <nodejs>
                    <enabled>true</enabled>
                </nodejs>
                <yarn>
                    <enabled>true</enabled>
                </yarn>
            </php>
        </default>
    </scopes>
</config>
`

// The rename has to reach every copy of the old name, and there are more of
// them than there look to be. Missing one is silent in the worst way: the key
// no longer exists, the renderer answers it as false, node stops being
// installed, and the failure surfaces much later as a build with no npm.
func TestV398_ReachesEveryCopy(t *testing.T) {
	execDir, registry, projectDir := installation(t)

	installConfig := filepath.Join(execDir, "config.xml")
	defaults := filepath.Join(execDir, "projects", "config.xml")
	registryConfig := filepath.Join(registry, "config.xml")
	projectConfig := filepath.Join(projectDir, ".madock", "config.xml")

	for _, path := range []string{installConfig, defaults, registryConfig, projectConfig} {
		write(t, path, oldSpelling)
	}

	// The registry has to know where the project is, or the migration cannot
	// find the two files that live with the source.
	write(t, registryConfig, strings.Replace(oldSpelling,
		"<platform>custom</platform>",
		"<platform>custom</platform>\n            <path>"+projectDir+"</path>", 1))

	// A template the project copied out of madock and edited. Nothing else
	// would ever fix this one.
	override := filepath.Join(projectDir, ".madock", "docker", "php", "Dockerfile")
	write(t, override, "{{{- if .php.nodejs.enabled}}}\nRUN echo node\n{{{- end}}}\n"+
		"{{{- if .php.yarn.enabled}}}\nRUN echo yarn\n{{{- end}}}\n")

	V398()

	for _, path := range []string{installConfig, defaults, registryConfig, projectConfig} {
		body := read(t, path)
		if strings.Contains(body, "<nodejs>\n                    <enabled>") {
			t.Errorf("%s still carries the old key:\n%s", path, body)
		}
		parsed := configs.ParseXmlFile(path)
		if got := parsed["scopes/default/nodejs/embedded/enabled"]; got != "true" {
			t.Errorf("%s: nodejs/embedded is %q, want \"true\"", path, got)
		}
		if got := parsed["scopes/default/nodejs/yarn/enabled"]; got != "true" {
			t.Errorf("%s: nodejs/yarn/enabled is %q, want \"true\"", path, got)
		}
		if _, still := parsed["scopes/default/php/nodejs/enabled"]; still {
			t.Errorf("%s: the old key is still there beside the new one", path)
		}
	}

	tmpl := read(t, override)
	if strings.Contains(tmpl, ".php.nodejs.enabled") || strings.Contains(tmpl, ".php.yarn.enabled") {
		t.Errorf("a project's own template still asks for the old key:\n%s", tmpl)
	}
	if !strings.Contains(tmpl, ".nodejs.embedded.enabled") {
		t.Errorf("the template was not migrated:\n%s", tmpl)
	}
}

// A project can carry several scopes, and a rename that fixed only `default`
// would leave the others reading a key that no longer exists — answered as
// false, with nothing said about it.
func TestV398_MigratesEveryScope(t *testing.T) {
	execDir, registry, _ := installation(t)

	write(t, filepath.Join(registry, "config.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <php><nodejs><enabled>true</enabled></nodejs></php>
        </default>
        <staging>
            <php><nodejs><enabled>false</enabled></nodejs></php>
        </staging>
    </scopes>
</config>
`)

	V398()

	parsed := configs.ParseXmlFile(filepath.Join(registry, "config.xml"))
	if got := parsed["scopes/default/nodejs/embedded/enabled"]; got != "true" {
		t.Errorf("default scope: %q", got)
	}
	if got := parsed["scopes/staging/nodejs/embedded/enabled"]; got != "false" {
		t.Errorf("staging scope: %q — a second scope was left behind", got)
	}
	_ = execDir
}

// The installation's config.xml can hold a hand-written top-level key: since
// 3.9.6 that is how a devops turns project:remove back on. A migration that
// rewrites the file must put it back, or the guard silently re-arms the next
// time anything is renamed.
func TestV398_KeepsWhatItDidNotComeFor(t *testing.T) {
	execDir, _, _ := installation(t)

	installConfig := filepath.Join(execDir, "config.xml")
	write(t, installConfig, `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <allow_destructive_commands>true</allow_destructive_commands>
    <scopes>
        <default>
            <db><password>not-to-be-touched</password></db>
            <php><nodejs><enabled>true</enabled></nodejs></php>
        </default>
    </scopes>
</config>
`)

	V398()

	parsed := configs.ParseXmlFile(installConfig)
	if got := parsed["allow_destructive_commands"]; got != "true" {
		t.Errorf("the top-level guard key is %q after a migration that never asked about it", got)
	}
	if got := parsed["scopes/default/db/password"]; got != "not-to-be-touched" {
		t.Errorf("an unrelated setting came back as %q", got)
	}
}

// Nothing to do must mean no diff. A migration that rewrites every config on
// every upgrade produces a churn of changes nobody can read, and on a
// project-local file that churn lands in somebody's repository.
func TestV398_LeavesUntouchedFilesAlone(t *testing.T) {
	execDir, registry, _ := installation(t)

	already := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <nodejs>
                <embedded>
                    <enabled>true</enabled>
                </embedded>
            </nodejs>
        </default>
    </scopes>
</config>
`
	path := filepath.Join(registry, "config.xml")
	write(t, path, already)

	V398()

	if after := read(t, path); after != already {
		t.Errorf("a file with nothing to migrate was rewritten:\n%s", after)
	}
	_ = execDir
}

// If somebody has already set the new name, theirs is the deliberate answer.
// The old key must go rather than be left to argue with it.
func TestV398_NewNameWins(t *testing.T) {
	execDir, registry, _ := installation(t)

	path := filepath.Join(registry, "config.xml")
	write(t, path, `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <php><nodejs><enabled>false</enabled></nodejs></php>
            <nodejs><embedded><enabled>true</enabled></embedded></nodejs>
        </default>
    </scopes>
</config>
`)

	V398()

	parsed := configs.ParseXmlFile(path)
	if got := parsed["scopes/default/nodejs/embedded/enabled"]; got != "true" {
		t.Errorf("the deliberate value was overwritten with %q", got)
	}
	if _, still := parsed["scopes/default/php/nodejs/enabled"]; still {
		t.Error("the old key survived beside the new one")
	}
	_ = execDir
}

// V320 wrote `language=php` over whatever a project actually was.
//
// Its guard asked rawConf["language"], but ParseXmlFile returns keys as they
// sit in the file — "scopes/default/language" — so the lookup never found one,
// the guard always passed, and a nodejs project became a php one. Measured: it
// sent `madock cli` into a php container that does not exist and cost half an
// hour looking for a defect in the service resolver.
//
// It only fires when the migrations run at all, which is why it hid: an
// installation with a current recorded version never reaches it, and a fresh
// one — the only safe way to try a new build — reaches it every time.
func TestV320_DoesNotOverwriteALanguageTheProjectAlreadyStates(t *testing.T) {
	_, registry, projectDir := installation(t)

	write(t, filepath.Join(registry, "config.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <platform>custom</platform>
            <language>nodejs</language>
            <path>`+projectDir+`</path>
        </default>
    </scopes>
</config>
`)

	projectConfig := filepath.Join(projectDir, ".madock", "config.xml")
	write(t, projectConfig, `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <platform>custom</platform>
            <language>nodejs</language>
        </default>
    </scopes>
</config>
`)

	// The broken path reads the *working directory*, not the registry's `path`,
	// and pointing at it is what makes this test catch anything: without it the
	// run stops at an earlier guard and passes against the defect.
	t.Setenv("MADOCK_RUN_DIR", projectDir)
	configs.CleanCache()

	V320()

	for _, path := range []string{filepath.Join(registry, "config.xml"), projectConfig} {
		if got := configs.ParseXmlFile(path)["scopes/default/language"]; got != "nodejs" {
			t.Errorf("%s: language became %q — the project was rewritten as something it is not", path, got)
		}
	}
}

// A project that genuinely states no language still gets one, or the migration
// stops doing the job it exists for.
func TestV320_StillFillsInAMissingLanguage(t *testing.T) {
	_, registry, projectDir := installation(t)

	write(t, filepath.Join(registry, "config.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <platform>magento2</platform>
            <path>`+projectDir+`</path>
        </default>
    </scopes>
</config>
`)

	V320()

	if got := configs.ParseXmlFile(filepath.Join(registry, "config.xml"))["scopes/default/language"]; got != "php" {
		t.Errorf("language is %q, want \"php\" — the backfill stopped working", got)
	}
}
