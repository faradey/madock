package project

import (
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Run these with -count=1, always:
//
//	go test -count=1 ./src/helper/configs/aruntime/project/...
//
// Go caches a package's test result until one of its .go files changes. These
// tests read their input from docker/ — plain data files the cache knows
// nothing about — so editing a template and re-running reports `ok (cached)`
// and proves nothing. Measured: a deliberately reversed dev/start preference
// passed a cached run and failed immediately with -count=1. The pre-push hook
// passes the flag for this reason.
//
// updateGolden rewrites the expected files instead of comparing against them:
//
//	go test -count=1 ./src/helper/configs/aruntime/project/... -run Golden -update
//
// Review the diff it produces like any other change. A golden file that was
// updated without being read is worse than no test — it records whatever the
// code does today and calls it correct.
var updateGolden = flag.Bool("update", false, "rewrite golden files")

// Golden tests render a project's whole docker configuration and compare it,
// file by file, against a committed copy.
//
// They exist because most of what breaks in madock breaks here. Of the eight
// defects found in the week these were written, four were generation: a Node
// entrypoint that preferred `dev` where a server needs `start`, a Dockerfile
// that was never written for a Node service on a non-PHP project, a config
// change that did not reach the generated files until a rebuild, and an
// OpenSearch default pointing at a tag that does not exist. All four are the
// kind of thing a diff of the rendered output shows immediately.
//
// The point of a golden file over an assertion is that it fails on changes
// nobody thought to assert. An assertion answers the question you asked; a
// golden file answers the one you did not.
type goldenCase struct {
	name      string
	overrides map[string]string
}

func goldenCases() []goldenCase {
	return []goldenCase{
		{
			// The default shape: Magento on PHP with MariaDB and OpenSearch.
			name: "magento2-php",
		},
		{
			// Two databases. The credentials of the second one are its own —
			// reading them from db/* is a defect this pins.
			name: "magento2-db2",
			overrides: map[string]string{
				"db2/enabled":       "true",
				"db2/root_password": "second_root",
				"db2/user":          "second_user",
				"db2/password":      "second_pw",
				"db2/database":      "second",
			},
		},
		{
			// PostgreSQL instead of MySQL: a different db service and a
			// different client in every command that talks to it.
			name: "magento2-postgres",
			overrides: map[string]string{
				"db/type":       "postgresql",
				"db/repository": "postgres",
				"db/version":    "16",
			},
		},
		{
			// Mail turned off. sendmail_path used to be written whatever the
			// configuration said, so with mailpit disabled every mail() call
			// went to a port nobody was listening on and failed silently. The
			// generated Dockerfile must simply not touch it.
			name: "magento2-no-sendmail",
			overrides: map[string]string{
				"php/sendmail/enabled": "false",
			},
		},
		{
			// With an envelope sender configured. Without one msmtp refuses a
			// plain mail() call, and the argument has to be absent rather than
			// empty when the address is not set — which is why it is rendered
			// whole rather than as a value.
			name: "magento2-sendmail-from",
			overrides: map[string]string{
				"php/sendmail/from": "shop@example.com",
			},
		},
		{
			// The sandbox shape — no language at all, so the main service is
			// "app" and there is no PHP container.
			name: "custom-none",
			overrides: map[string]string{
				"platform":    "custom",
				"language":    "none",
				"php/enabled": "false",
				"app/enabled": "true",
			},
		},
		{
			// Node as the main service. Its Dockerfile comes from the language
			// template, which carries cron where the service template does not.
			name: "custom-nodejs",
			overrides: map[string]string{
				"platform":       "custom",
				"language":       "nodejs",
				"php/enabled":    "false",
				"nodejs/enabled": "true",
			},
		},
		{
			// A Node service beside a language that is not Node. The compose
			// file renders the service on nodejs/enabled alone, and its
			// Dockerfile used to be written only for PHP projects — so this
			// case existed as a compose service pointing at a missing file.
			name: "custom-none-with-nodejs",
			overrides: map[string]string{
				"platform":       "custom",
				"language":       "none",
				"php/enabled":    "false",
				"app/enabled":    "true",
				"nodejs/enabled": "true",
			},
		},
		{
			// A php platform of its own. Pins the fastcgi HTTPS parameter, which
			// used to be the constant `on`: the application then believed every
			// request was secure, including one that arrived over plain http at the
			// published project port, and built https URLs and secure cookies for
			// it. Shopware and Sylius hardcoded SERVER_PORT 443 beside it.
			name: "woocommerce-php",
			overrides: map[string]string{
				"platform": "woocommerce",
				"language": "php",
			},
		},
		{
			// PHP switched on beside a language that is not PHP. The nginx
			// templates used to include a snippet per enabled runtime, so this
			// shape rendered two server blocks on the same listen and
			// server_name: nginx kept the php one and warned about the other,
			// and the route to the app answered 404 from a document root that
			// has no index.php. The front door follows the language, so the
			// only server block here proxies to app.
			name: "custom-none-with-php",
			overrides: map[string]string{
				"platform":    "custom",
				"language":    "none",
				"php/enabled": "true",
				"app/enabled": "true",
			},
		},
		{
			// What a Node project on a server looks like: production, and a
			// named script rather than whatever package.json happens to have.
			name: "custom-nodejs-production",
			overrides: map[string]string{
				"platform":            "custom",
				"language":            "nodejs",
				"php/enabled":         "false",
				"nodejs/enabled":      "true",
				"nodejs/env":          "production",
				"nodejs/script":       "docker-start",
				"nodejs/script_type":  "command",
				"nodejs/browser_libs": "true",
			},
		},
	}
}

func TestGoldenGeneratedConfig(t *testing.T) {
	for _, testCase := range goldenCases() {
		t.Run(testCase.name, func(t *testing.T) {
			projectName := "golden"
			env := setupTestEnvironmentWith(t, projectName, "golden.test", testCase.overrides)

			MakeConf(projectName)

			rendered := collectGenerated(t, filepath.Join(env.execDir, "aruntime", "projects", projectName), env)
			if len(rendered) == 0 {
				t.Fatal("MakeConf produced no files")
			}

			goldenDir := filepath.Join("testdata", "golden", testCase.name)
			if *updateGolden {
				writeGolden(t, goldenDir, rendered)
				return
			}
			compareGolden(t, goldenDir, rendered)
		})
	}
}

// collectGenerated reads every generated file, with the machine-specific parts
// replaced so the same output is expected on any machine.
func collectGenerated(t *testing.T, root string, env *testEnv) map[string]string {
	t.Helper()

	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// The runtime directory holds symlinks to the project source, the
		// composer home and ~/.ssh. Following them would pull the whole
		// working tree into the comparison.
		if d.Type()&fs.ModeSymlink != 0 || d.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		files[filepath.ToSlash(rel)] = normalise(string(content), env)
		return nil
	})
	if err != nil {
		t.Fatalf("reading generated files: %v", err)
	}
	return files
}

var (
	// Only a ports mapping, and only where compose writes one: a list entry.
	// Matching every `"NNNN:` in the file swallowed `user: "1001:1001"` on a
	// Linux runner, where a uid happens to be four digits.
	publishedPort = regexp.MustCompile(`- "\d{4,5}:`)

	// uid and gid are normalised where they are *used*, not wherever their
	// digits appear. Replacing the values themselves looked simpler and was
	// wrong twice over: a macOS gid of 20 rewrote unrelated numbers inside the
	// Grafana dashboards ("2093" became "<GID>93"), and on a CI runner where
	// uid == gid the second replacement found nothing left to replace, so
	// `groupmod -g` came out as `<UID>`. Both produced golden files that could
	// only ever match the machine that wrote them.
	composeUser = regexp.MustCompile(`user: "\d+:\d+"`)
	usermodUid  = regexp.MustCompile(`usermod -u \d+`)
	groupmodGid = regexp.MustCompile(`groupmod -g \d+`)
	chownPair   = regexp.MustCompile(`chown (-R )?\d+:\d+`)

	// The loader URL carries the host architecture, so an arm64 laptop and an
	// amd64 runner render different files from the same template.
	loaderArch = regexp.MustCompile(`ioncube_loaders_lin_[\w-]+\.tar\.gz`)
)

// normalise removes what differs between machines and runs.
//
// Ports are allocated by probing the host for something free, so they depend on
// what else is listening; uid, gid and the CPU architecture come from whoever
// runs the tests. None of that is what these tests are about — the shape of the
// rendered file is.
func normalise(content string, env *testEnv) string {
	content = strings.ReplaceAll(content, env.execDir, "<EXEC_DIR>")
	content = strings.ReplaceAll(content, env.runDir, "<RUN_DIR>")
	content = publishedPort.ReplaceAllString(content, `- "<PORT>:`)
	content = composeUser.ReplaceAllString(content, `user: "<UID>:<GID>"`)
	content = usermodUid.ReplaceAllString(content, "usermod -u <UID>")
	content = groupmodGid.ReplaceAllString(content, "groupmod -g <GID>")
	content = chownPair.ReplaceAllString(content, "chown ${1}<UID>:<GID>")
	content = loaderArch.ReplaceAllString(content, "ioncube_loaders_lin_<ARCH>.tar.gz")

	return content
}

func writeGolden(t *testing.T, goldenDir string, rendered map[string]string) {
	t.Helper()

	if err := os.RemoveAll(goldenDir); err != nil {
		t.Fatalf("clearing %s: %v", goldenDir, err)
	}
	for name, content := range rendered {
		path := filepath.Join(goldenDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("creating %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	t.Logf("wrote %d golden files to %s — read the diff before committing", len(rendered), goldenDir)
}

func compareGolden(t *testing.T, goldenDir string, rendered map[string]string) {
	t.Helper()

	expected := map[string]string{}
	err := filepath.WalkDir(goldenDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(goldenDir, path)
		expected[filepath.ToSlash(rel)] = string(content)
		return nil
	})
	if err != nil {
		t.Fatalf("no golden files in %s (%v) — run with -update", goldenDir, err)
	}

	for name, want := range expected {
		got, rendered_ok := rendered[name]
		if !rendered_ok {
			t.Errorf("%s is no longer generated", name)
			continue
		}
		if got != want {
			t.Errorf("%s differs from the golden copy:\n%s", name, firstDifference(want, got))
		}
	}

	for name := range rendered {
		if _, known := expected[name]; !known {
			t.Errorf("%s is generated but has no golden copy — new output, or a file that moved", name)
		}
	}
}

// firstDifference reports the first differing line with a little context, which
// is what a reader needs; a full diff of a 200-line compose file is not.
func firstDifference(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")

	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		wantLine := ""
		if i < len(wantLines) {
			wantLine = wantLines[i]
		}
		gotLine := ""
		if i < len(gotLines) {
			gotLine = gotLines[i]
		}
		if wantLine != gotLine {
			return "  line " + itoa(i+1) + "\n    want: " + wantLine + "\n    got:  " + gotLine
		}
	}
	return "  (files differ only in trailing content)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

// TestNormalise pins the thing the golden files depend on and nothing else
// checks: that two different machines normalise to the same text.
//
// Both cases here are real. macOS gives a gid of 20, which the old
// value-substitution normaliser found inside unrelated numbers; a GitHub runner
// gives uid == gid, where replacing the uid first left the gid with nothing to
// match. Either one produced golden files that only the machine that wrote them
// could reproduce.
func TestNormalise(t *testing.T) {
	env := &testEnv{execDir: "/exec", runDir: "/run"}

	// The same rendered file as three machines would write it: an arm64 laptop
	// (uid 501, gid 20), an amd64 runner (uid 1001, gid 1001), and a Linux
	// desktop (uid 1000, gid 1000).
	render := func(uid, gid, arch string) string {
		return strings.Join([]string{
			`    user: "` + uid + `:` + gid + `"`,
			`      - "34500:80"`,
			`RUN usermod -u ` + uid + ` -o www-data && groupmod -g ` + gid + ` -o www-data`,
			`RUN mkdir /var/www/.npm && chown ` + uid + `:` + gid + ` /var/www/.npm`,
			`    && chown -R ` + uid + `:` + gid + ` /var/www`,
			`    && curl -o ioncube.tar.gz http://x/ioncube_loaders_lin_` + arch + `.tar.gz \\`,
			`RUN sed -i 's/session.cookie_lifetime = 0/session.cookie_lifetime = 2592000/g' php.ini`,
			`        "uid": "PBFA97CFB590B2093"`,
			`  ingestion_burst_size_mb: 20`,
		}, "\n")
	}

	mac := normalise(render("501", "20", "aarch64"), env)
	ci := normalise(render("1001", "1001", "x86-64"), env)
	linux := normalise(render("1000", "1000", "x86-64"), env)

	if mac != ci {
		t.Errorf("macOS and CI normalise differently:\n%s\n---\n%s", mac, ci)
	}
	if ci != linux {
		t.Errorf("two Linux machines normalise differently:\n%s\n---\n%s", ci, linux)
	}

	// The numbers that are not ids must survive, and the ids must not.
	for _, want := range []string{
		`user: "<UID>:<GID>"`,
		`- "<PORT>:80"`,
		"usermod -u <UID> -o www-data && groupmod -g <GID> -o www-data",
		"chown <UID>:<GID> /var/www/.npm",
		"chown -R <UID>:<GID> /var/www",
		"ioncube_loaders_lin_<ARCH>.tar.gz",
		"session.cookie_lifetime = 2592000",
		`"uid": "PBFA97CFB590B2093"`,
		"ingestion_burst_size_mb: 20",
	} {
		if !strings.Contains(mac, want) {
			t.Errorf("normalised output is missing %q:\n%s", want, mac)
		}
	}
}
