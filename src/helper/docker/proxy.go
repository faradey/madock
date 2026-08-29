package docker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	configs2 "github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/configs/aruntime/nginx"
	"github.com/faradey/madock/v4/src/helper/configs/aruntime/project"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
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
		// Output, not CombinedOutput: docker's warnings go to stderr and would
		// otherwise be handed to a JSON reader as if they were data.
		result, err := cmd.Output()
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

	ctxPath := paths.MakeDirsByPath(paths.CtxDir())

	// The certificate covers every project at once, so adding a project or
	// editing its hosts invalidates it. It used to be issued only on the very
	// first start of the proxy — behind both "the proxy is down" and "there is
	// no conf-cache marker" — while proxy.conf was regenerated every time. A
	// new project was therefore routed immediately and served over HTTPS with a
	// certificate that did not name it, and a renamed host kept the old name.
	//
	// Checked before the branch on purpose: the gap is in both halves. `restart`
	// stops the last project, which takes the proxy down with it, so the case
	// where the proxy is down and the marker still exists is the common one.
	// Reissuing costs a second or two and happens only when the host set really
	// changed.
	certRefreshed := false
	if !nginx.SslCertCoversCurrentHosts(ctxPath) {
		nginx.GenerateSslCert(ctxPath, false)
		certRefreshed = true
	}

	if doNeedRunAruntime {
		// Proxy is not running (first start / proxy:rebuild did Down) → bring it up.
		CreateProxyNetwork()

		if !paths.IsFileExist(confCache) {
			if !certRefreshed {
				nginx.GenerateSslCert(ctxPath, false)
			}

			// force is proxy:rebuild, and only there is looking for a newer
			// image the point. Every other start reaches here through
			// UpNginx, which passes false: the proxy image only has to be
			// present, and asking the registry about one already on the
			// machine is what this used to do on the first start of every
			// installation.
			dockerComposePull([]string{"compose", "-f", proxyCompose}, force)

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
	} else if certRefreshed || (newHash != "" && newHash != readProxyHash(hashCache)) {
		// Proxy is already running and its config changed (a project rebuild/clone
		// regenerated proxy.conf) → reload in place so other projects stay up
		// (zero-downtime). reload re-parses the full config: routing, upstreams
		// (re-resolves container DNS) and certs.
		if err := ReloadNginx(); err == nil {
			// Persist the applied hash and restore the conf-cache marker the
			// rebuild removed (so MakeConf resumes caching) only when the reload
			// command actually ran — if the exec failed (proxy missing / docker
			// error) we must not record the config as applied or we'd never retry.
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

// ReloadNginx reloads the nginx configuration in the running proxy and returns
// the exec error so callers can avoid recording a config as "applied" when the
// command could not even run (e.g. the proxy container is missing or docker
// errored). It targets the compose SERVICE name ("nginx") rather than a
// container name: the proxy template sets no container_name, so Compose v2 names
// the container "<project>-nginx-<index>" (aruntime-nginx-1) — a hardcoded
// container name would not match. `-T` disables TTY allocation (non-interactive).
//
// Note: `nginx -s reload` returns 0 even when the new config is rejected (nginx
// logs the error and keeps the old config and workers live), so a non-nil error
// here means the exec itself failed, not that the config was bad.
// IsProxyRunning reports whether the aruntime nginx container is up.
func IsProxyRunning() bool {
	composeFile := paths.ProxyDockerCompose()
	if !paths.IsFileExist(composeFile) {
		return false
	}
	// Output, not CombinedOutput — see the note in status.getContainerStatus:
	// compose writes deprecation warnings to stderr, and mixing them into the
	// data is how a JSON reader ends up complaining about English prose.
	out, err := exec.Command("docker", "compose", "-f", composeFile, "ps", "--format", "json").Output()
	if err != nil {
		return false
	}

	return isProxyRunning(out)
}

// ProxyLogs shows the proxy's own container logs — the only place the reason for
// a 502 is written down. nginx logs to /var/log/nginx/{access,error}.log, and the
// image symlinks both to the container's stdout and stderr, so `docker compose
// logs` carries them.
//
// Returns the captured output when not following, so the caller can tell an empty
// log apart from a log full of errors instead of printing nothing and letting it
// read as "no problems".
func ProxyLogs(service string, follow bool, tail string) (string, error) {
	composeFile := paths.ProxyDockerCompose()
	if !paths.IsFileExist(composeFile) {
		return "", fmt.Errorf("the proxy has never been set up: %s does not exist", composeFile)
	}

	command := []string{"compose", "-f", composeFile, "logs", "--no-color"}
	if tail != "" {
		command = append(command, "--tail", tail)
	}
	if follow {
		command = append(command, "--follow")
	}
	if service != "" {
		command = append(command, service)
	}

	cmd := exec.Command("docker", command...)
	if follow {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return "", cmd.Run()
	}

	out, err := cmd.CombinedOutput()

	return string(out), err
}

func ReloadNginx() error {
	composeFile := paths.ProxyDockerCompose()
	cmd := exec.Command("docker", "compose", "-f", composeFile, "exec", "-T", "nginx", "nginx", "-s", "reload")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out), err)
	}
	return err
}

// CreateProxyNetwork creates the shared network for proxy and project services
func CreateProxyNetwork() {
	// Ignore error if network already exists
	cmd := exec.Command("docker", "network", "create", "--driver", "bridge", "madock-proxy")
	_ = cmd.Run()
}
