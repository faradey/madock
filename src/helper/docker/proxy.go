package docker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/configs/aruntime/nginx"
	"github.com/faradey/madock/v3/src/helper/configs/aruntime/project"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
)

// UpNginx starts the nginx proxy container
func UpNginx(projectName string) {
	UpNginxWithBuild(projectName, false)
}

// UpNginxWithBuild starts the nginx proxy container with optional rebuild
func UpNginxWithBuild(projectName string, force bool) {
	if !paths.IsFileExist(paths.GetRunDirPath() + "/.madock/config.xml") {
		configs2.SetParam(configs2.MadockLevelConfigCode, "path", paths.GetRunDirPath(), "default", configs2.MadockLevelConfigCode)
	}
	nginx.MakeConf(projectName)
	project.MakeConf(projectName)
	projectConf := configs2.GetProjectConfig(projectName)
	doNeedRunAruntime := true
	proxyCompose := paths.ProxyDockerCompose()
	if paths.IsFileExist(proxyCompose) {
		cmd := exec.Command("docker", "compose", "-f", proxyCompose, "ps", "--format", "json")
		result, err := cmd.CombinedOutput()
		if err != nil {
			logger.Println(err, result)
		} else if isProxyRunning(result) {
			doNeedRunAruntime = false
		}
	}

	if projectConf["proxy/enabled"] != "true" {
		return
	}

	confCache := paths.CacheDir() + "/conf-cache"
	// Hash of the generated proxy.conf actually applied to the running proxy.
	// nginx.MakeConf above regenerates proxy.conf on every call (so a freshly
	// added/started project shows up); comparing the fresh hash against the
	// last-applied one lets us reload only when the config truly changed.
	newHash := proxyConfHash()
	hashCache := paths.CacheDir() + "/proxy-conf-hash"

	if doNeedRunAruntime {
		// Proxy is not running (first start / proxy:rebuild did Down) → bring it up.
		CreateProxyNetwork()

		ctxPath := paths.MakeDirsByPath(paths.CtxDir())
		if !paths.IsFileExist(confCache) {
			nginx.GenerateSslCert(ctxPath, false)

			dockerComposePull([]string{"compose", "-f", proxyCompose})

			err := os.WriteFile(confCache, []byte("config cache"), 0755)
			if err != nil {
				logger.Fatal(err)
			}
		}

		command := []string{"compose", "-f", proxyCompose, "up", "--no-deps", "-d"}
		if force {
			command = append(command, "--build", "--force-recreate")
		}
		cmd := exec.Command("docker", command...)
		attachOutput(cmd)
		if err := cmd.Run(); err != nil {
			logger.Println(err)
		} else {
			// Record the applied config only on a successful up; otherwise the
			// proxy isn't actually running this config and the next run must retry.
			writeProxyHash(hashCache, newHash)
		}
	} else if newHash != "" && newHash != readProxyHash(hashCache) {
		// Proxy is already running and its config changed (a project rebuild/clone
		// regenerated proxy.conf) → reload in place so other projects stay up
		// (zero-downtime). reload re-parses the full config: routing, upstreams
		// (re-resolves container DNS) and certs.
		if err := ReloadNginx(); err == nil {
			// Persist the applied hash and restore the conf-cache marker the
			// rebuild removed (so MakeConf resumes caching) only when the reload
			// actually took — a rejected config keeps the old one live, so we must
			// not record it as applied or we'd never retry.
			writeProxyHash(hashCache, newHash)
			if !paths.IsFileExist(confCache) {
				if err := os.WriteFile(confCache, []byte("config cache"), 0755); err != nil {
					logger.Fatal(err)
				}
			}
		}
	}
}

// proxyConfHash returns the SHA-256 hex of the generated proxy.conf (the file
// mounted into the proxy container). Empty string if it doesn't exist yet.
func proxyConfHash() string {
	data, err := os.ReadFile(paths.CtxDir() + "/proxy.conf")
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func readProxyHash(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func writeProxyHash(path, hash string) {
	if hash == "" {
		return
	}
	if err := os.WriteFile(path, []byte(hash), 0644); err != nil {
		logger.Println(err)
	}
}

// isProxyRunning reports whether the aruntime-nginx container is in the
// "running" state, based on `docker compose ps --format json` output. The
// output is either NDJSON (one object per line, newer compose) or a single
// JSON array (older compose); both are handled. A present-but-stopped
// container must NOT count as running, otherwise we'd try to reload a dead
// proxy instead of bringing it up.
func isProxyRunning(psOutput []byte) bool {
	type psEntry struct {
		Service string `json:"Service"`
		Name    string `json:"Name"`
		State   string `json:"State"`
	}
	isRunning := func(e psEntry) bool {
		return e.State == "running" &&
			(strings.Contains(e.Service, "aruntime-nginx") || strings.Contains(e.Name, "aruntime-nginx"))
	}

	for _, line := range strings.Split(string(psOutput), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			var entries []psEntry
			if err := json.Unmarshal([]byte(line), &entries); err == nil {
				for _, e := range entries {
					if isRunning(e) {
						return true
					}
				}
			}
			continue
		}
		var e psEntry
		if err := json.Unmarshal([]byte(line), &e); err == nil && isRunning(e) {
			return true
		}
	}
	return false
}

// DownNginx stops and removes the nginx proxy container
func DownNginx(force bool) {
	composeFile := paths.ProxyDockerCompose()
	if paths.IsFileExist(composeFile) {
		command := "down"
		if force {
			command = "kill"
		}
		cmd := exec.Command("docker", "compose", "-f", composeFile, command)
		attachOutput(cmd)
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}

// StopNginx stops the nginx proxy container
func StopNginx(force bool) {
	composeFile := paths.ProxyDockerCompose()
	if paths.IsFileExist(composeFile) {
		command := "stop"
		if force {
			command = "kill"
		}
		cmd := exec.Command("docker", "compose", "-f", composeFile, command)
		attachOutput(cmd)
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}

// ReloadNginx reloads the nginx configuration. Returns the exec error so
// callers can avoid recording a config as "applied" when the reload failed
// (e.g. nginx rejected the new config and kept running the old one).
func ReloadNginx() error {
	err := ContainerExec("aruntime-nginx", "", false, "nginx", "-s", "reload")
	if err != nil {
		fmt.Println(err)
	}
	return err
}

// CreateProxyNetwork creates the shared network for proxy and project services
func CreateProxyNetwork() {
	// Ignore error if network already exists
	cmd := exec.Command("docker", "network", "create", "--driver", "bridge", "madock-proxy")
	_ = cmd.Run()
}
