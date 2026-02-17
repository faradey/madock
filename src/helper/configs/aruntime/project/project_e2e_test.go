//go:build e2e

package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/ports"
)

// setupE2EEnvironment creates a test environment with redis enabled on top of the base setup.
func setupE2EEnvironment(t *testing.T, projectName, hostName string) *testEnv {
	t.Helper()

	env := setupTestEnvironment(t, projectName, hostName)

	// Enable redis in the project config via SaveInFile (merges into existing XML)
	projectConfigPath := filepath.Join(env.execDir, "projects", projectName, "config.xml")
	configs.SaveInFile(projectConfigPath, map[string]string{
		"redis/enabled": "true",
	}, "default")

	// Clean cache and reset ports after config change
	configs.CleanCache()
	ports.ResetRegistry()

	return env
}

// waitForContainer polls a docker exec command until it succeeds or times out.
// Returns stdout on success.
func waitForContainer(t *testing.T, container string, check []string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	var lastOutput string

	for time.Now().Before(deadline) {
		args := append([]string{"exec", container}, check...)
		cmd := exec.Command("docker", args...)
		out, err := cmd.CombinedOutput()
		lastOutput = string(out)
		if err == nil {
			return lastOutput
		}
		lastErr = err
		time.Sleep(5 * time.Second)
	}

	t.Fatalf("Container %s not ready after %v. Last error: %v, output: %s", container, timeout, lastErr, lastOutput)
	return ""
}

func TestE2E_Magento2_ContainersStart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	projectName := "e2etest"
	env := setupE2EEnvironment(t, projectName, "e2etest.test")

	// Generate all docker config files
	MakeConf(projectName)

	runtimeDir := filepath.Join(env.execDir, "aruntime", "projects", projectName)

	// Create the external docker network (ignore error if it already exists)
	exec.Command("docker", "network", "create", "madock-proxy").Run()

	// Create directories that containers expect to mount
	mountDirs := []string{
		filepath.Join(runtimeDir, "composer"),
		filepath.Join(runtimeDir, "ssh"),
		filepath.Join(runtimeDir, "ctx", "scripts"),
	}
	for _, d := range mountDirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("Failed to create mount dir %s: %v", d, err)
		}
	}
	// Create empty known_hosts file for SSH mount
	knownHosts := filepath.Join(runtimeDir, "ssh", "known_hosts")
	if err := os.WriteFile(knownHosts, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create known_hosts: %v", err)
	}

	// Compose files
	composeFile := filepath.Join(runtimeDir, "docker-compose.yml")
	overrideFile := filepath.Join(runtimeDir, "docker-compose.override.yml")

	// Register cleanup BEFORE starting containers so it always runs
	t.Cleanup(func() {
		downCmd := exec.Command("docker", "compose",
			"-f", composeFile, "-f", overrideFile,
			"down", "-v", "--rmi", "local")
		downCmd.Dir = runtimeDir
		downOut, _ := downCmd.CombinedOutput()
		t.Logf("docker compose down output:\n%s", string(downOut))

		// Remove the external network (ignore error if other projects use it)
		exec.Command("docker", "network", "rm", "madock-proxy").Run()
	})

	// Start containers
	t.Log("Starting docker compose up --build ...")
	upCmd := exec.Command("docker", "compose",
		"-f", composeFile, "-f", overrideFile,
		"up", "--build", "--force-recreate", "--no-deps", "-d")
	upCmd.Dir = runtimeDir
	upOut, err := upCmd.CombinedOutput()
	t.Logf("docker compose up output:\n%s", string(upOut))
	if err != nil {
		t.Fatalf("docker compose up failed: %v", err)
	}

	// Container name prefix: madock_e2etest-{service}-1
	prefix := "madock_" + projectName

	// Wait for and verify DB (MariaDB 11.4 uses mariadb-admin; -h 127.0.0.1 forces TCP auth)
	t.Log("Waiting for DB container...")
	dbOutput := waitForContainer(t, prefix+"-db-1",
		[]string{"mariadb-admin", "-u", "root", "-ppassword", "-h", "127.0.0.1", "ping"},
		3*time.Minute)
	if !strings.Contains(dbOutput, "alive") {
		t.Errorf("DB check: expected 'alive' in ping output, got: %s", dbOutput)
	}

	// Wait for and verify PHP
	t.Log("Waiting for PHP container...")
	phpOutput := waitForContainer(t, prefix+"-php-1",
		[]string{"php", "-v"},
		3*time.Minute)
	if !strings.Contains(phpOutput, "8.4") {
		t.Errorf("PHP check: expected version containing '8.4', got: %s", phpOutput)
	}

	// Wait for and verify OpenSearch
	t.Log("Waiting for OpenSearch container...")
	osOutput := waitForContainer(t, prefix+"-opensearch-1",
		[]string{"curl", "-s", "http://localhost:9200"},
		3*time.Minute)
	if !strings.Contains(osOutput, "2.19.0") {
		t.Errorf("OpenSearch check: expected version '2.19.0', got: %s", osOutput)
	}

	// Wait for and verify Redis
	t.Log("Waiting for Redis container...")
	redisOutput := waitForContainer(t, prefix+"-redisdb-1",
		[]string{"redis-cli", "ping"},
		3*time.Minute)
	if !strings.Contains(redisOutput, "PONG") {
		t.Errorf("Redis check: expected 'PONG', got: %s", redisOutput)
	}

	// Verify Nginx container is running
	t.Log("Checking Nginx container status...")
	inspectCmd := exec.Command("docker", "inspect",
		"--format", "{{.State.Status}}", prefix+"-nginx-1")
	inspectOut, err := inspectCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to inspect nginx container: %v, output: %s", err, string(inspectOut))
	}
	nginxStatus := strings.TrimSpace(string(inspectOut))
	if nginxStatus != "running" {
		t.Errorf("Nginx container status: expected 'running', got: %s", nginxStatus)
	}

	t.Log("All containers verified successfully")
}
