package node

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"node"},
		Handler:  Execute,
		Help:     "Execute node command",
		Category: "general",
	})
}

func Execute() {
	flag := cli.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	service := "nodejs"

	service, user, workdir := cli.GetEnvForUserServiceWorkdir(service, "www-data", projectConf["workdir"])

	isMutation := checkMutation(projectName, flag, service, user, workdir, projectConf)
	if !isMutation {
		err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, service), user, true, "bash", "-c", "cd "+workdir+" && "+flag)
		if err != nil {
			logger.Fatal(err)
		}
	}
}

func checkMutation(projectName, command, service, user, workdir string, projectConf map[string]string) bool {
	if projectConf["platform"] == "magento2" && strings.Contains(command, "grunt") && (strings.Contains(command, "exec:") || strings.Contains(command, "refresh")) {
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
				for key, theme := range themes.X {
					err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, service), user, true, "bash", "-c", "cd "+workdir+" && grunt --force clean:"+key)
					if err != nil {
						logger.Fatal(err)
					}
					files := theme.(map[string]interface{})["files"]
					joinedFiles := ""
					for _, file := range files.([]interface{}) {
						joinedFiles += file.(string) + " "
					}
					err = docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), user, true, "bash", "-c", "cd "+workdir+" && php bin/magento dev:source-theme:deploy "+joinedFiles+" --type=less --locale="+theme.(map[string]interface{})["locale"].(string)+" --area="+theme.(map[string]interface{})["area"].(string)+" --theme="+theme.(map[string]interface{})["name"].(string))
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
