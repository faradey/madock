package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/testenv"
)

func TestMakeConfMagento2Integration(t *testing.T) {
	env := testenv.Setup(t, "testproject", "magento248.test")

	MakeConf(env.ProjectName)

	runtimeDir := filepath.Join(env.ExecDir, "aruntime", "projects", env.ProjectName)
	ctxDir := filepath.Join(runtimeDir, "ctx")

	// Verify expected files exist
	expectedFiles := []string{
		filepath.Join(runtimeDir, "docker-compose.yml"),
		filepath.Join(runtimeDir, "docker-compose.override.yml"),
		filepath.Join(ctxDir, "nginx.Dockerfile"),
		filepath.Join(ctxDir, "nginx.conf"),
		filepath.Join(ctxDir, "php.Dockerfile"),
		filepath.Join(ctxDir, "db.Dockerfile"),
		filepath.Join(ctxDir, "my.cnf"),
		filepath.Join(ctxDir, "opensearch.Dockerfile"),
		filepath.Join(ctxDir, "redis.Dockerfile"),
	}

	for _, f := range expectedFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("Expected file not generated: %s", filepath.Base(f))
		}
	}

	// Verify php.Dockerfile contains PHP 8.4
	assertFileContains(t, filepath.Join(ctxDir, "php.Dockerfile"), "8.4")

	// Verify db.Dockerfile contains mariadb and version 11.4
	assertFileContains(t, filepath.Join(ctxDir, "db.Dockerfile"), "mariadb")
	assertFileContains(t, filepath.Join(ctxDir, "db.Dockerfile"), "11.4")

	// Verify my.cnf contains [mysqld] and MariaDB optimizations
	assertFileContains(t, filepath.Join(ctxDir, "my.cnf"), "[mysqld]")
	assertFileContains(t, filepath.Join(ctxDir, "my.cnf"), "optimizer_switch")

	// Verify docker-compose.yml has key services
	assertFileContains(t, filepath.Join(runtimeDir, "docker-compose.yml"), "php")
	assertFileContains(t, filepath.Join(runtimeDir, "docker-compose.yml"), "nginx")
	assertFileContains(t, filepath.Join(runtimeDir, "docker-compose.yml"), "db")
}

func TestMakeConfMagento2_NginxHostConfig(t *testing.T) {
	env := testenv.Setup(t, "hostproject", "magento248.test")

	MakeConf(env.ProjectName)

	ctxDir := filepath.Join(env.ExecDir, "aruntime", "projects", env.ProjectName, "ctx")
	nginxConf := filepath.Join(ctxDir, "nginx.conf")

	assertFileContains(t, nginxConf, "magento248.test")
}

func TestMakeConfMagento2_DockerComposeServices(t *testing.T) {
	env := testenv.Setup(t, "svcproject", "magento248.test")

	MakeConf(env.ProjectName)

	runtimeDir := filepath.Join(env.ExecDir, "aruntime", "projects", env.ProjectName)
	composeFile := filepath.Join(runtimeDir, "docker-compose.yml")

	content, err := os.ReadFile(composeFile)
	if err != nil {
		t.Fatalf("Failed to read docker-compose.yml: %v", err)
	}
	composeStr := string(content)

	// Check that key services are defined
	services := []string{"php", "nginx", "db"}
	for _, svc := range services {
		if !strings.Contains(composeStr, svc) {
			t.Errorf("docker-compose.yml missing service %q", svc)
		}
	}

	// Check that Dockerfile references point to ctx/
	if !strings.Contains(composeStr, "ctx/") {
		t.Error("docker-compose.yml should reference Dockerfiles in ctx/")
	}
}

// assertFileContains checks that a file exists and contains the given substring.
func assertFileContains(t *testing.T, path, substr string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Cannot read %s: %v", filepath.Base(path), err)
		return
	}
	if !strings.Contains(string(content), substr) {
		t.Errorf("%s does not contain %q", filepath.Base(path), substr)
	}
}

// TestMakeConf_DbServiceEnabledByDefault renders a project config that does not
// mention db/enabled at all — the shape every project created before the flag
// existed has on disk. The embedded default must fill it in, otherwise
// evaluateCondition sees an unsubstituted {{{db/enabled}}} placeholder, treats
// it as false, and the database container silently disappears.
func TestMakeConf_DbServiceEnabledByDefault(t *testing.T) {
	env := testenv.Setup(t, "dbdefaultproject", "dbdefault.test")

	MakeConf(env.ProjectName)

	composeStr := readCompose(t, env)

	if !strings.Contains(composeStr, "\n  db:\n") {
		t.Error("db service missing from docker-compose.yml when db/enabled is not set")
	}
	if !strings.Contains(composeStr, "\n  dbdata:\n") {
		t.Error("dbdata volume missing from docker-compose.yml when db/enabled is not set")
	}
}

// TestMakeConf_DbServiceDisabled checks the flag actually removes the container
// and its volume, and that nothing else goes with them.
func TestMakeConf_DbServiceDisabled(t *testing.T) {
	env := testenv.Setup(t, "dbdisabledproject", "dbdisabled.test")

	projectConfigPath := filepath.Join(env.ExecDir, "projects", env.ProjectName, "config.xml")
	configs.SaveInFile(projectConfigPath, map[string]string{"db/enabled": "false"}, "default")
	configs.CleanCache()

	MakeConf(env.ProjectName)

	composeStr := readCompose(t, env)

	if strings.Contains(composeStr, "\n  db:\n") {
		t.Error("db service still rendered when db/enabled=false")
	}
	// The dbdata volume stays declared even with the database off: every other
	// entry in the volumes section is optional, and an empty "volumes:" makes
	// compose fail with "volumes must be a mapping".
	if !strings.Contains(composeStr, "\n  dbdata:\n") {
		t.Error("dbdata volume declaration dropped; volumes section can now render empty")
	}

	// The rest of the stack must be untouched.
	for _, svc := range []string{"\n  php:\n", "\n  nginx:\n"} {
		if !strings.Contains(composeStr, svc) {
			t.Errorf("service %q disappeared along with the database", strings.TrimSpace(svc))
		}
	}
}

// TestMakeConf_MemcachedDisabledByDefault keeps the new service opt-in. It is
// included in every platform's compose file, so a wrong default would add an
// idle container to every existing project on the next rebuild.
func TestMakeConf_MemcachedDisabledByDefault(t *testing.T) {
	env := testenv.Setup(t, "memcdefaultproject", "memcdefault.test")

	MakeConf(env.ProjectName)

	composeStr := readCompose(t, env)

	if strings.Contains(composeStr, "\n  memcached:\n") {
		t.Error("memcached service rendered although the service is off by default")
	}
	if strings.Contains(readPhpDockerfile(t, env), "-memcached") {
		t.Error("php-memcached installed into the PHP image although the service is off")
	}
}

// TestMakeConf_MemcachedEnabled renders the service from a project config that
// sets only memcached/enabled — the shape service:enable writes. Image tag, cache
// size and connection limit must come from the embedded defaults; an unresolved
// placeholder would land in docker-compose.yml verbatim and break the stack.
func TestMakeConf_MemcachedEnabled(t *testing.T) {
	env := testenv.Setup(t, "memcenabledproject", "memcenabled.test")

	projectConfigPath := filepath.Join(env.ExecDir, "projects", env.ProjectName, "config.xml")
	configs.SaveInFile(projectConfigPath, map[string]string{"memcached/enabled": "true"}, "default")
	configs.CleanCache()

	MakeConf(env.ProjectName)

	composeStr := readCompose(t, env)

	if !strings.Contains(composeStr, "\n  memcached:\n") {
		t.Fatal("memcached service missing from docker-compose.yml when memcached/enabled=true")
	}
	if !strings.Contains(composeStr, "image: memcached:") {
		t.Error("memcached image not resolved from the embedded repository/version defaults")
	}
	if !strings.Contains(composeStr, `command: ["-m", "256", "-c", "1024"]`) {
		t.Error("memcached command not resolved from the embedded memory/max_connections defaults")
	}
	if strings.Contains(composeStr, "{{{memcached/") {
		t.Error("unresolved memcached placeholder left in docker-compose.yml")
	}

	// The container alone is useless to PHP without the extension.
	if !strings.Contains(readPhpDockerfile(t, env), "-memcached") {
		t.Error("php-memcached missing from the PHP image while the service is enabled")
	}
}

func readPhpDockerfile(t *testing.T, env *testenv.Env) string {
	t.Helper()
	dockerfile := filepath.Join(env.ExecDir, "aruntime", "projects", env.ProjectName, "ctx", "php.Dockerfile")
	content, err := os.ReadFile(dockerfile)
	if err != nil {
		t.Fatalf("Failed to read php.Dockerfile: %v", err)
	}
	return string(content)
}

func readCompose(t *testing.T, env *testenv.Env) string {
	t.Helper()
	composeFile := filepath.Join(env.ExecDir, "aruntime", "projects", env.ProjectName, "docker-compose.yml")
	content, err := os.ReadFile(composeFile)
	if err != nil {
		t.Fatalf("Failed to read docker-compose.yml: %v", err)
	}
	return string(content)
}

// TestMakeConf_VolumesNeverEmpty renders the leanest project the config allows —
// no database, no second database, no search engine, no grafana — and checks the
// volumes section still has an entry. Every entry there except dbdata is
// optional, so gating dbdata as well lets the section render as a bare
// "volumes:" and compose refuses the file with "volumes must be a mapping".
func TestMakeConf_VolumesNeverEmpty(t *testing.T) {
	env := testenv.Setup(t, "novolumesproject", "novolumes.test")

	projectConfigPath := filepath.Join(env.ExecDir, "projects", env.ProjectName, "config.xml")
	configs.SaveInFile(projectConfigPath, map[string]string{
		"db/enabled":                   "false",
		"db2/enabled":                  "false",
		"search/opensearch/enabled":    "false",
		"search/elasticsearch/enabled": "false",
		"search/meilisearch/enabled":   "false",
		"grafana/enabled":              "false",
	}, "default")
	configs.CleanCache()

	MakeConf(env.ProjectName)

	composeStr := readCompose(t, env)

	idx := strings.Index(composeStr, "\nvolumes:\n")
	if idx == -1 {
		t.Fatal("volumes section missing from docker-compose.yml")
	}

	rest := composeStr[idx+len("\nvolumes:\n"):]
	firstLine := rest
	if nl := strings.Index(rest, "\n"); nl != -1 {
		firstLine = rest[:nl]
	}
	if !strings.HasPrefix(firstLine, "  ") || strings.TrimSpace(firstLine) == "" {
		t.Errorf("volumes section is empty; compose would fail with \"volumes must be a mapping\". Next line: %q", firstLine)
	}
}
