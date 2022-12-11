package commands

import (
	"fmt"
	"log"
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

func Remote(flag, option string) {
	if flag == "sync" {
		projectConfig := configs.GetCurrentProjectConfig()
		if option == "media" {
			ssh.SyncMedia(projectConfig["SSH_SITE_ROOT_PATH"])
		} else if option == "db" {
			conn := ssh.Connect(projectConfig["SSH_AUTH_TYPE"], projectConfig["SSH_KEY_PATH"], projectConfig["SSH_PASSWORD"], projectConfig["SSH_HOST"], projectConfig["SSH_PORT"], projectConfig["SSH_USERNAME"])
			ssh.DbDump(conn, projectConfig["SSH_SITE_ROOT_PATH"])
		} else if option == "file" {
			ssh.SyncFile(projectConfig["SSH_SITE_ROOT_PATH"])
		}
	} else {
		log.Fatal("The specified parameters were not found.")
	}
}

func Proxy(flag string) {
	if !configs.IsHasNotConfig() {
		builder.PrepareConfigs()
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
		builder.Down()
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

func Composer(flag string) {
	builder.Composer(flag)
}

func Compress() {
	compress.Zip()
}

func Uncompress() {
	compress.Unzip()
}

func Debug(flag string) {
	configPath := paths.GetExecDirPath() + "/projects/" + paths.GetRunDirName() + "/env.txt"
	if flag == "on" {
		configs.SetParam(configPath, "XDEBUG_ENABLED", "true")
	} else if flag == "off" {
		configs.SetParam(configPath, "XDEBUG_ENABLED", "false")
	} else {
		log.Fatal("The specified parameters were not found.")
	}
	builder.UpWithBuild()
}

func Info() {
	scripts.MagentoInfo()
}

func Cron(flag string) {
	if flag == "on" || flag == "off" {
		builder.Cron(flag, true)
	} else {
		log.Fatal("The specified parameters were not found.")
	}
}

func Bash(flag string) {
	containerName := "php"
	if flag != "" {
		containerName = flag
	}

	builder.Bash(containerName)
}

func CleanCache() {
	builder.CleanCache()
}

func SetEnvOption() {
	if attr.Options.Hosts {
		if len(attr.Options.Args) > 0 {
			configPath := paths.GetExecDirPath() + "/projects/" + paths.GetRunDirName() + "/env.txt"
			configs.SetParam(configPath, "HOSTS", strings.Join(attr.Options.Args, " "))
		} else {
			fmtc.ErrorLn("Specify at least one domain")
		}
	}
}

func ShowEnv() {
	configPath := paths.GetExecDirPath() + "/projects/" + paths.GetRunDirName() + "/env.txt"
	lines := configs.GetAllLines(configPath)
	for _, ln := range lines {
		fmt.Println(ln)
	}
}

func Node(flag string) {
	builder.Node(flag)
}

func Logs(flag string) {
	containerName := "php"
	if flag != "" {
		containerName = flag
	}
	builder.Logs(containerName)
}

func IsNotDefine() {
	fmtc.ErrorLn("The command is not defined. Run 'madock help' to invoke help")
}

func Ssl(flag string) {
	if flag == "rebuild" {
		builder.SslRebuild()
	}
}
