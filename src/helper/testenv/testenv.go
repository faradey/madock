// Package testenv builds a madock installation in a temporary directory so that
// generation can be tested against the real templates, and compares the result
// against committed golden files.
//
// It is a package of its own rather than a set of test helpers because two
// packages need it and Go does not let one package import another's test files:
// a project's generated configuration is rendered by the project package, while
// the shared proxy configuration is rendered by the nginx package — which imports
// project, so the test cannot live there either.
//
// Nothing in madock imports this, so testing never reaches the shipped binary.
package testenv

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/ports"
)

// ProjectRoot locates the madock project root by walking up from the current test file.
func ProjectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot find project root (go.mod)")
		}
		dir = parent
	}
}

// Env holds paths for the test environment and cleanup function.
type Env struct {
	ExecDir     string
	RunDir      string
	ProjectName string
}

// Setup creates a temp directory structure that MakeConf can work with.
// It symlinks docker/ and scripts/ from the real project, copies config.xml,
// and creates a project config via SaveInFile.
func Setup(t *testing.T, projectName, hostName string) *Env {
	t.Helper()
	return SetupWith(t, projectName, hostName, nil)
}

// SetupWith is Setup with the project config
// opened up: overrides are applied on top of the Magento defaults below, so a
// test can describe a different platform, language or database without
// restating the ninety keys it does not care about.
func SetupWith(t *testing.T, projectName, hostName string, overrides map[string]string) *Env {
	t.Helper()
	realRoot := ProjectRoot(t)

	tmpDir, err := os.MkdirTemp("", "madock-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	execDir := filepath.Join(tmpDir, "exec")
	runDir := filepath.Join(tmpDir, "run")

	// Create directory structure
	dirs := []string{
		execDir,
		filepath.Join(execDir, "projects", projectName, "docker", "ctx"),
		filepath.Join(execDir, "aruntime", "projects", projectName, "ctx"),
		filepath.Join(execDir, "cache"),
		filepath.Join(execDir, "scripts"),
		runDir,
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", d, err)
		}
	}

	// Symlink docker/ from real project
	if err := os.Symlink(filepath.Join(realRoot, "docker"), filepath.Join(execDir, "docker")); err != nil {
		t.Fatalf("Failed to symlink docker/: %v", err)
	}

	// Copy config.xml (global config template)
	srcConfig, err := os.ReadFile(filepath.Join(realRoot, "config.xml"))
	if err != nil {
		t.Fatalf("Failed to read config.xml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(execDir, "config.xml"), srcConfig, 0644); err != nil {
		t.Fatalf("Failed to write config.xml: %v", err)
	}

	// Set env vars
	t.Setenv("MADOCK_EXEC_DIR", execDir)
	t.Setenv("MADOCK_RUN_DIR", runDir)

	// Clean config cache so the new env vars are picked up
	configs.CleanCache()

	writeProjectConfig(execDir, runDir, projectName, hostName, overrides)

	// Reset global port registry so it re-reads from the new execDir
	ports.ResetRegistry()

	return &Env{
		ExecDir:     execDir,
		RunDir:      runDir,
		ProjectName: projectName,
	}
}

// AddProject puts a second project into an installation Setup already made.
//
// One project is enough for everything generated *inside* a project, and that is
// what every fixture covered until now. The shared proxy is the exception: it is
// one file describing every project on the machine, so the interesting part of
// it — that two projects get their own upstreams, their own server blocks and
// their own ports, and that neither overwrites the other — cannot be rendered
// from a single project at all.
//
// The second project lives in the same installation and gets a run directory of
// its own. MADOCK_RUN_DIR moves with it, because that is how madock decides
// which project it is being asked about.
func AddProject(t *testing.T, env *Env, projectName, hostName string, overrides map[string]string) *Env {
	t.Helper()

	runDir := filepath.Join(filepath.Dir(env.RunDir), "run-"+projectName)
	for _, dir := range []string{
		runDir,
		filepath.Join(env.ExecDir, "projects", projectName, "docker", "ctx"),
		filepath.Join(env.ExecDir, "aruntime", "projects", projectName, "ctx"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	t.Setenv("MADOCK_RUN_DIR", runDir)
	configs.CleanCache()

	writeProjectConfig(env.ExecDir, runDir, projectName, hostName, overrides)

	return &Env{
		ExecDir:     env.ExecDir,
		RunDir:      runDir,
		ProjectName: projectName,
	}
}

// writeProjectConfig writes one project's configuration, defaults first and the
// caller's overrides on top.
func writeProjectConfig(execDir, runDir, projectName, hostName string, overrides map[string]string) {
	projectConfigPath := filepath.Join(execDir, "projects", projectName, "config.xml")
	projectConfigData := map[string]string{
		"platform":                                  "magento2",
		"language":                                  "php",
		"path":                                      runDir,
		"php/enabled":                               "true",
		"php/version":                               "8.4",
		"php/composer/version":                      "2",
		"php/xdebug/version":                        "3.4.4",
		"php/xdebug/remote_host":                    "host.docker.internal",
		"php/xdebug/ide_key":                        "PHPSTORM",
		"php/xdebug/enabled":                        "false",
		"php/xdebug/mode":                           "debug",
		"php/ioncube/enabled":                       "false",
		"nodejs/embedded/enabled":                        "false",
		"timezone":                                  "UTC",
		"workdir":                                   "/var/www/html",
		"public_dir":                                "pub",
		"composer_dir":                              "",
		"db/repository":                             "mariadb",
		"db/version":                                "11.4",
		"db/root_password":                          "password",
		"db/user":                                   "magento",
		"db/password":                               "magento",
		"db/database":                               "magento",
		"db/phpmyadmin/enabled":                     "false",
		"db2/enabled":                               "false",
		"search/engine":                             "OpenSearch",
		"search/elasticsearch/enabled":              "false",
		"search/elasticsearch/version":              "8.17.6",
		"search/elasticsearch/repository":           "elasticsearch",
		"search/opensearch/enabled":                 "true",
		"search/opensearch/version":                 "2.19.0",
		"search/opensearch/repository":              "opensearchproject/opensearch",
		"search/opensearch/dashboard/enabled":       "false",
		"search/opensearch/dashboard/repository":    "opensearchproject/opensearch-dashboards",
		"search/elasticsearch/dashboard/enabled":    "false",
		"search/elasticsearch/dashboard/repository": "kibana",
		"redis/enabled":                             "false",
		"redis/repository":                          "redis",
		"redis/version":                             "8.0",
		"valkey/enabled":                            "false",
		"valkey/repository":                         "valkey/valkey",
		"valkey/version":                            "8.1.3",
		"rabbitmq/enabled":                          "false",
		"rabbitmq/repository":                       "rabbitmq",
		"rabbitmq/version":                          "4.1",
		"nodejs/enabled":                            "false",
		"nodejs/repository":                         "node",
		"nodejs/version":                            "18.15.0",
		"nodejs/major_version":                      "18",
		"nodejs/yarn/enabled":                       "false",
		"cron/enabled":                              "false",
		"nginx/ssl/enabled":                         "true",
		"nginx/http/version":                        "http2",
		"nginx/run_type":                            "website",
		"nginx/default_host_first_level":            ".test",
		"nginx/port/unsecure":                       "80",
		"nginx/port/secure":                         "443",
		"nginx/port/internal":                       "80",
		"nginx/interface_ip":                        "",
		"os/name":                                   "ubuntu",
		"os/version":                                "22.04",
		"container_name_prefix":                     "madock_",
		"restart_policy":                            "no",
		"proxy/enabled":                             "true",
		"proxy/timeout/connect":                     "60",
		"proxy/timeout/send":                        "300",
		"proxy/timeout/read":                        "300",
		"proxy/gzip/enabled":                        "true",
		"proxy/rate_limit/enabled":                  "true",
		"proxy/rate_limit/rate":                     "50",
		"proxy/rate_limit/burst":                    "200",
		"proxy/conn_limit/enabled":                  "true",
		"proxy/conn_limit/per_ip":                   "100",
		"proxy/max_body_size":                       "128M",
		"isolation/enabled":                         "false",
		"varnish/enabled":                           "false",
		"grafana/enabled":                           "false",
		"claude/enabled":                            "false",
		"claude/nodejs_repository":                  "node",
		"claude/nodejs_version":                     "22.19.0",
		"ssh/auth_type":                             "key",
		"ssh/port":                                  "22",
		"magento/admin_user":                        "admin",
		"magento/admin_password":                    "admin123",
		"magento/admin_first_name":                  "admin",
		"magento/admin_last_name":                   "admin",
		"magento/admin_email":                       "admin@admin.com",
		"magento/admin_frontname":                   "admin",
		"magento/locale":                            "en_US",
		"magento/currency":                          "USD",
		"magento/timezone":                          "America/Chicago",
		"magento/cloud/enabled":                     "false",
		"magento/mftf/enabled":                      "false",
		"magento/n98magerun/enabled":                "false",
	}

	if hostName != "" {
		projectConfigData["nginx/hosts/base/name"] = hostName
	}

	for key, value := range overrides {
		projectConfigData[key] = value
	}

	configs.SaveInFile(projectConfigPath, projectConfigData, "default")

	// Clean cache again after writing config
	configs.CleanCache()
}

// Collect reads every generated file, with the machine-specific parts
// replaced so the same output is expected on any machine.
func Collect(t *testing.T, root string, env *Env) map[string]string {
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
		files[filepath.ToSlash(rel)] = Normalise(string(content), env)
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

	// The same uid and gid again, in the shape the non-PHP language images use:
	// a build argument rather than a usermod call. Nothing rendered one until
	// the platform fixtures arrived — python and ruby build this way — so the
	// first run of those on a Linux runner failed on a macOS uid of 501 against
	// its own 1001, which is the whole reason a golden is normalised at all.
	dockerfileUid = regexp.MustCompile(`ARG MADOCK_UID=\d+`)
	dockerfileGid = regexp.MustCompile(`ARG MADOCK_GID=\d+`)

	// An allocated port inside a URL. `publishedPort` above sees only a compose
	// list entry, and a project that has to tell its own application where it
	// lives — Medusa's storefront does, in NEXT_PUBLIC_BASE_URL — writes the
	// port into an environment variable instead, where nothing was masking it.
	//
	// `localhost` only, deliberately. The proxy's golden masks by *position* —
	// the first distinct port becomes <PORT1>, the second <PORT2>, so a block
	// pointing at the wrong upstream still shows as a difference — and it does
	// that after this runs. Masking `host.docker.internal:NNNNN` here left it
	// nothing to number, and the proxy fixture failed on <PORT1> against <PORT>.
	urlPort = regexp.MustCompile(`localhost:1[78]\d{3}`)
)

// Normalise removes what differs between machines and runs.
//
// Ports are allocated by probing the host for something free, so they depend on
// what else is listening; uid, gid and the CPU architecture come from whoever
// runs the tests. None of that is what these tests are about — the shape of the
// rendered file is.
func Normalise(content string, env *Env) string {
	content = strings.ReplaceAll(content, env.ExecDir, "<EXEC_DIR>")
	content = strings.ReplaceAll(content, env.RunDir, "<RUN_DIR>")
	content = publishedPort.ReplaceAllString(content, `- "<PORT>:`)
	content = composeUser.ReplaceAllString(content, `user: "<UID>:<GID>"`)
	content = usermodUid.ReplaceAllString(content, "usermod -u <UID>")
	content = groupmodGid.ReplaceAllString(content, "groupmod -g <GID>")
	content = chownPair.ReplaceAllString(content, "chown ${1}<UID>:<GID>")
	content = loaderArch.ReplaceAllString(content, "ioncube_loaders_lin_<ARCH>.tar.gz")
	content = dockerfileUid.ReplaceAllString(content, "ARG MADOCK_UID=<UID>")
	content = dockerfileGid.ReplaceAllString(content, "ARG MADOCK_GID=<GID>")
	content = urlPort.ReplaceAllString(content, "localhost:<PORT>")

	return content
}

func WriteGolden(t *testing.T, goldenDir string, rendered map[string]string) {
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

func CompareGolden(t *testing.T, goldenDir string, rendered map[string]string) {
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
