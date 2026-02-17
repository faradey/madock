package project

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/ports"
)

// findProjectRoot locates the madock project root by walking up from the current test file.
func findProjectRoot(t *testing.T) string {
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

// testEnv holds paths for the test environment and cleanup function.
type testEnv struct {
	execDir     string
	runDir      string
	projectName string
}

// setupTestEnvironment creates a temp directory structure that MakeConf can work with.
// It symlinks docker/ and scripts/ from the real project, copies config.xml,
// and creates a project config via SaveInFile.
func setupTestEnvironment(t *testing.T, projectName, hostName string) *testEnv {
	t.Helper()
	realRoot := findProjectRoot(t)

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

	// Create project config via SaveInFile with Magento 2.4.8 settings
	projectConfigPath := filepath.Join(execDir, "projects", projectName, "config.xml")
	projectConfigData := map[string]string{
		"platform":                     "magento2",
		"language":                     "php",
		"path":                         runDir,
		"php/enabled":                  "true",
		"php/version":                  "8.4",
		"php/composer/version":         "2",
		"php/xdebug/version":           "3.4.4",
		"php/xdebug/remote_host":       "host.docker.internal",
		"php/xdebug/ide_key":           "PHPSTORM",
		"php/xdebug/enabled":           "false",
		"php/xdebug/mode":              "debug",
		"php/ioncube/enabled":          "false",
		"php/nodejs/enabled":           "false",
		"timezone":                     "Europe/Kiev",
		"workdir":                      "/var/www/html",
		"public_dir":                   "pub",
		"composer_dir":                 "",
		"db/repository":                "mariadb",
		"db/version":                   "11.4",
		"db/root_password":             "password",
		"db/user":                      "magento",
		"db/password":                  "magento",
		"db/database":                  "magento",
		"db/phpmyadmin/enabled":        "false",
		"db2/enabled":                  "false",
		"search/engine":                "OpenSearch",
		"search/elasticsearch/enabled": "false",
		"search/elasticsearch/version": "8.17.6",
		"search/elasticsearch/repository": "elasticsearch",
		"search/opensearch/enabled":    "true",
		"search/opensearch/version":    "2.19.0",
		"search/opensearch/repository": "opensearchproject/opensearch",
		"search/opensearch/dashboard/enabled":    "false",
		"search/opensearch/dashboard/repository": "opensearchproject/opensearch-dashboards",
		"search/elasticsearch/dashboard/enabled":    "false",
		"search/elasticsearch/dashboard/repository": "kibana",
		"redis/enabled":                "false",
		"redis/repository":             "redis",
		"redis/version":                "8.0",
		"valkey/enabled":               "false",
		"valkey/repository":            "valkey/valkey",
		"valkey/version":               "8.1.3",
		"rabbitmq/enabled":             "false",
		"rabbitmq/repository":          "rabbitmq",
		"rabbitmq/version":             "4.1",
		"nodejs/enabled":               "false",
		"nodejs/repository":            "node",
		"nodejs/version":               "18.15.0",
		"nodejs/major_version":         "18",
		"nodejs/yarn/enabled":          "false",
		"cron/enabled":                 "false",
		"nginx/ssl/enabled":            "true",
		"nginx/http/version":           "http2",
		"nginx/run_type":               "website",
		"nginx/default_host_first_level": ".test",
		"nginx/port/unsecure":          "80",
		"nginx/port/secure":            "443",
		"nginx/port/internal":          "80",
		"nginx/interface_ip":           "",
		"os/name":                      "ubuntu",
		"os/version":                   "22.04",
		"container_name_prefix":        "madock_",
		"restart_policy":               "no",
		"proxy/enabled":                "true",
		"proxy/timeout/connect":        "60",
		"proxy/timeout/send":           "300",
		"proxy/timeout/read":           "300",
		"proxy/gzip/enabled":           "true",
		"proxy/rate_limit/enabled":     "true",
		"proxy/rate_limit/rate":        "1000",
		"proxy/rate_limit/burst":       "2000",
		"isolation/enabled":            "false",
		"varnish/enabled":              "false",
		"grafana/enabled":              "false",
		"claude/enabled":               "false",
		"claude/nodejs_repository":     "node",
		"claude/nodejs_version":        "22.19.0",
		"ssh/auth_type":                "key",
		"ssh/port":                     "22",
		"magento/admin_user":           "admin",
		"magento/admin_password":       "admin123",
		"magento/admin_first_name":     "admin",
		"magento/admin_last_name":      "admin",
		"magento/admin_email":          "admin@admin.com",
		"magento/admin_frontname":      "admin",
		"magento/locale":               "en_US",
		"magento/currency":             "USD",
		"magento/timezone":             "America/Chicago",
		"magento/cloud/enabled":        "false",
		"magento/mftf/enabled":         "false",
		"magento/n98magerun/enabled":   "false",
	}

	if hostName != "" {
		projectConfigData["nginx/hosts/base/name"] = hostName
	}

	configs.SaveInFile(projectConfigPath, projectConfigData, "default")

	// Clean cache again after writing config
	configs.CleanCache()
	// Reset global port registry so it re-reads from the new execDir
	ports.ResetRegistry()

	return &testEnv{
		execDir:     execDir,
		runDir:      runDir,
		projectName: projectName,
	}
}

func TestMakeConfMagento2Integration(t *testing.T) {
	env := setupTestEnvironment(t, "testproject", "magento248.test")

	MakeConf(env.projectName)

	runtimeDir := filepath.Join(env.execDir, "aruntime", "projects", env.projectName)
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
	env := setupTestEnvironment(t, "hostproject", "magento248.test")

	MakeConf(env.projectName)

	ctxDir := filepath.Join(env.execDir, "aruntime", "projects", env.projectName, "ctx")
	nginxConf := filepath.Join(ctxDir, "nginx.conf")

	assertFileContains(t, nginxConf, "magento248.test")
}

func TestMakeConfMagento2_DockerComposeServices(t *testing.T) {
	env := setupTestEnvironment(t, "svcproject", "magento248.test")

	MakeConf(env.projectName)

	runtimeDir := filepath.Join(env.execDir, "aruntime", "projects", env.projectName)
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
