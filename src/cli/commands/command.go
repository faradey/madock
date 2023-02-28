package commands

import (
	"fmt"
	"strings"

	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/compress"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/faradey/madock/src/docker/scripts"
	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/ssh"
)

func RemoteSyncDb() {
	projectConfig := configs.GetCurrentProjectConfig()
	conn := ssh.Connect(projectConfig["SSH_AUTH_TYPE"], projectConfig["SSH_KEY_PATH"], projectConfig["SSH_PASSWORD"], projectConfig["SSH_HOST"], projectConfig["SSH_PORT"], projectConfig["SSH_USERNAME"])
	ssh.DbDump(conn, projectConfig["SSH_SITE_ROOT_PATH"], attr.Options.Name)
}

func RemoteSyncMedia() {
	projectConfig := configs.GetCurrentProjectConfig()
	ssh.SyncMedia(projectConfig["SSH_SITE_ROOT_PATH"])
}

func RemoteSyncFile() {
	projectConfig := configs.GetCurrentProjectConfig()
	ssh.SyncFile(projectConfig["SSH_SITE_ROOT_PATH"])
}

func Proxy(flag string) {
	if !configs.IsHasNotConfig() {
		if flag == "prune" {
			builder.DownNginx()
		} else if flag == "stop" {
			builder.StopNginx()
		} else if flag == "restart" {
			builder.StopNginx()
			builder.UpNginx()
		} else if flag == "start" {
			builder.UpNginx()
		} else if flag == "rebuild" {
			builder.DownNginx()
			builder.UpNginxWithBuild()
		}
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}

func Prune() {
	if !configs.IsHasNotConfig() {
		builder.Down(attr.Options.WithVolumes)
		if len(paths.GetActiveProjects()) == 0 {
			Proxy("stop")
		}
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}

func Magento(flag string) {
	builder.Magento(flag)
}

func Cloud(flag string) {
	projectConfig := configs.GetCurrentProjectConfig()
	flag = strings.Replace(flag, "$project", projectConfig["MAGENTOCLOUD_PROJECT_NAME"], -1)
	builder.Cloud(flag)
}

func Cli(flag string) {
	builder.Cli(flag)
}

func Composer(flag string) {
	builder.Composer(flag)
}

func Compress() {
	compress.Zip()
}

func Uncompress() {
	compress.Unzip()
}

func DebugEnable() {
	configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
	configs.SetParam(configPath, "XDEBUG_ENABLED", "true")
	builder.UpWithBuild()
}

func DebugDisable() {
	configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
	configs.SetParam(configPath, "XDEBUG_ENABLED", "false")
	builder.UpWithBuild()
}

func Info() {
	scripts.MagentoInfo()
}

func CronEnable() {
	builder.Cron(true, true)
}

func CronDisable() {
	builder.Cron(false, true)
}

func Bash() {
	containerName := "php"
	if len(attr.Options.Args) > 0 && attr.Options.Args[0] != "" {
		containerName = attr.Options.Args[0]
	}

	builder.Bash(containerName)
}

func CleanCache() {
	builder.CleanCache()
}

func SetEnvOption() {
	name := strings.ToUpper(attr.Options.Name)
	val := attr.Options.Value
	if len(name) > 0 && configs.IsOption(name) {
		configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
		configs.SetParam(configPath, name, val)
	}
}

func ShowEnv() {
	configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
	lines := configs.GetAllLines(configPath)
	for _, ln := range lines {
		fmt.Println(ln)
	}
}

func Node(flag string) {
	builder.Node(flag)
}

func Logs() {
	containerName := "php"
	if len(attr.Options.Args) > 0 && attr.Options.Args[0] != "" {
		containerName = attr.Options.Args[0]
	}
	builder.Logs(containerName)
}

func IsNotDefine() {
	fmtc.ErrorLn("The command is not defined. Run 'madock help' to invoke help")
}

func Ssl() {
	builder.SslRebuild()
}
