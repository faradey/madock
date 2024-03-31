package node

import (
	"encoding/json"
	"fmt"
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"os"
	"os/exec"
	"strings"
)

func Execute() {
	flag := cli.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	service := "nodejs"

	service, user, workdir := cli.GetEnvForUserServiceWorkdir(service, "www-data", projectConf["workdir"])

	isMutation := checkMutation(projectName, flag, service, user, workdir, projectConf)
	if !isMutation {
		cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, service), "bash", "-c", "cd "+workdir+" && "+flag)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}
	}
}

func checkMutation(projectName, command, service, user, workdir string, projectConf map[string]string) bool {
	if projectConf["platform"] == "magento2" && strings.Contains(command, "grunt") && (strings.Contains(command, "exec:") || strings.Contains(command, "refresh")) {
		cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, service), "bash", "-c", "cd "+workdir+" && grunt --force clean")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}

		if paths.IsFileExist(paths.GetRunDirPath() + "/grunt-config.json") {
			type GruntConfig struct {
				Themes string `json:"themes"`
			}
			var gruntConfig GruntConfig
			data, err := os.ReadFile(paths.GetRunDirPath() + "/grunt-config.json")
			if err != nil {
				fmt.Println(err)
				return false
			}
			err = json.Unmarshal(data, &gruntConfig)
			if err != nil {
				fmt.Println(err)
				return false
			}
			if paths.IsFileExist(paths.GetRunDirPath() + "/" + gruntConfig.Themes + ".js") {
				data, err = os.ReadFile(paths.GetRunDirPath() + "/" + gruntConfig.Themes + ".js")
				if err != nil {
					fmt.Println(err)
					return false
				}
				data = []byte(strings.ReplaceAll(strings.Split(strings.Join(strings.Fields(string(data)), ""), "module.exports=")[1], "};", "}"))
				data = []byte(strings.ReplaceAll(string(data), "\"", ""))
				data = []byte(strings.ReplaceAll(string(data), "'", ""))
				data = []byte(strings.ReplaceAll(string(data), "{", "{\""))
				data = []byte(strings.ReplaceAll(string(data), ":", "\":\""))
				data = []byte(strings.ReplaceAll(string(data), ",", "\",\""))
				data = []byte(strings.ReplaceAll(string(data), ":\"{", ":{"))
				data = []byte(strings.ReplaceAll(string(data), "}", "\"}"))
				data = []byte(strings.ReplaceAll(string(data), "}\"", "}"))
				data = []byte(strings.ReplaceAll(string(data), "\":\"[", "\":[\""))
				data = []byte(strings.ReplaceAll(string(data), "]\"", "\"]"))
				type Themes struct {
					X map[string]interface{} `json:"-"`
				}
				type ThemeConfig struct {
					Area   string `json:"area"`
					Name   string `json:"name"`
					Locale string `json:"locale"`
					Files  string `json:"files"`
					Dsl    string `json:"dsl"`
				}
				fmt.Println(string(data))
				var themes Themes
				err = json.Unmarshal(data, &themes)
				if err != nil {
					fmt.Println(err)
					return false
				}
				err = json.Unmarshal(data, &themes.X)
				if err != nil {
					fmt.Println(err)
					return false
				}
				for _, theme := range themes.X {
					cmd = exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+workdir+" && php bin/magento dev:source-theme:deploy --type=less --locale="+theme.(map[string]interface{})["locale"].(string)+" --area="+theme.(map[string]interface{})["area"].(string)+" --theme="+theme.(map[string]interface{})["name"].(string))
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					err = cmd.Run()
					if err != nil {
						logger.Fatal(err)
					}
				}
				return true
			}
		}
	}
	return false
}
