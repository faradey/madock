package service

import (
	"log"
	"strings"

	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
)

func ServiceEnable() {
	if len(attr.Options.Args) > 0 {
		name := strings.ToLower(attr.Options.Args[0])
		if isService(name) {
			serviceName := strings.ToUpper(name) + "_ENABLED"
			projectName := paths.GetRunDirName()
			envFile := ""
			if !attr.Options.Global {
				envFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
			} else {
				envFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects") + "/config.txt"
			}
			configs.SetParam(envFile, serviceName, "true")
		}
	}
}

func ServiceDisable() {
	if len(attr.Options.Args) > 0 {
		name := strings.ToLower(attr.Options.Args[0])
		if isService(name) {
			serviceName := strings.ToUpper(name) + "_ENABLED"
			projectName := paths.GetRunDirName()
			envFile := ""
			if !attr.Options.Global {
				envFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
			} else {
				envFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects") + "/config.txt"
			}
			configs.SetParam(envFile, serviceName, "false")
		}
	}
}

func isService(name string) bool {
	upperName := strings.ToUpper(name)
	configData := configs.GetCurrentProjectConfig()

	for key := range configData {
		serviceName := strings.SplitN(key, "_ENABLED", 2)
		if serviceName[0] == upperName {
			return true
		}
	}

	log.Fatalln("The service \"" + name + "\" doesn't exist.")

	return false
}
