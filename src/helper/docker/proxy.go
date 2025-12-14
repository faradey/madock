package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/aruntime/nginx"
	"github.com/faradey/madock/src/helper/configs/aruntime/project"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
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
		} else {
			if len(result) > 100 && strings.Contains(string(result), "\"Command\"") && strings.Contains(string(result), "\"aruntime-nginx\"") {
				doNeedRunAruntime = false
			}
		}
	}

	confCache := paths.CacheDir() + "/conf-cache"
	if (!paths.IsFileExist(confCache) || doNeedRunAruntime) && projectConf["proxy/enabled"] == "true" {
		// Create shared network for proxy and services
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
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			logger.Println(err)
		}
	}
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
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
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
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}

// ReloadNginx reloads the nginx configuration
func ReloadNginx() {
	cmd := exec.Command("docker", "exec", "aruntime-nginx", "nginx", "-s", "reload")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}

// CreateProxyNetwork creates the shared network for proxy and project services
func CreateProxyNetwork() {
	// Ignore error if network already exists
	cmd := exec.Command("docker", "network", "create", "--driver", "bridge", "madock-proxy")
	_ = cmd.Run()
}
