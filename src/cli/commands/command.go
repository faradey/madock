package commands

import (
	"fmt"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/ssh"
	"log"
	"strings"
)

func Start() {
	if !configs.IsHasNotConfig() {
		fmtc.SuccessLn("Start containers in detached mode")
		builder.Start()
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}

func Stop() {
	builder.Stop()
}

func Restart() {
	Stop()
	Start()
}

func Rebuild() {
	if !configs.IsHasNotConfig() {
		fmtc.SuccessLn("Stop containers")
		builder.Down()
		fmtc.SuccessLn("Start containers in detached mode")
		builder.UpWithBuild()
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}

func Remote(flag, option string) {
	if flag == "--sync" {
		if option == "media" {
			projectConfig := configs.GetCurrentProjectConfig()
			conn := ssh.Connect(projectConfig["SSH_KEY_PATH"], projectConfig["SSH_HOST"], projectConfig["SSH_PORT"], projectConfig["SSH_USERNAME"])
			ssh.Sync(conn, projectConfig["SSH_SITE_ROOT_PATH"])
		}
	} else {
		log.Fatal("The specified parameters were not found.")
	}
}

func Prune(flag string) {
	if !configs.IsHasNotConfig() {
		if flag == "--all" {
			builder.DownAll()
		} else {
			builder.Down()
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

func Composer(flag string) {
	builder.Composer(flag)
}

func DB(flag, option string) {
	if flag == "--import" {
		builder.DbImport(option)
	} else if flag == "--export" {
		builder.DbExport()
	} else if flag == "--soft-clean" {
		builder.DbSoftClean()
	} else {
		log.Fatal("The specified parameters were not found.")
	}
}

func Debug(flag string) {
	configPath := paths.GetExecDirPath() + "/projects/" + paths.GetRunDirName() + "/env.txt"
	if flag == "--on" {
		configs.SetParam(configPath, "PHP_MODULE_XDEBUG", "true")
	} else if flag == "--off" {
		configs.SetParam(configPath, "PHP_MODULE_XDEBUG", "false")
	} else {
		log.Fatal("The specified parameters were not found.")
	}
	builder.UpWithBuild()
}

func Cron(flag string) {
	if flag == "--on" || flag == "--off" {
		builder.Cron(flag, true)
	} else {
		log.Fatal("The specified parameters were not found.")
	}
}

func Bash(flag, flag2, flag3 string) {
	containerName := "php"
	isRoot := false
	if flag == "--root" {
		isRoot = true
		if flag2 == "--name" {
			containerName = flag3
		}
	} else {
		if flag3 == "--root" {
			isRoot = true
		}
		if flag == "--name" {
			containerName = flag2
		}
	}

	builder.Bash(containerName, isRoot)
}

func SetEnvOption(flag string, flags []string) {
	if flag == "--hosts" {
		if len(flags) > 0 {
			configPath := paths.GetExecDirPath() + "/projects/" + paths.GetRunDirName() + "/env.txt"
			configs.SetParam(configPath, "HOSTS", strings.Join(flags, " "))
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

func Logs(flag, flag2 string) {
	containerName := "php"
	if flag == "--name" {
		containerName = flag2
	}
	builder.Logs(containerName)
}

func IsNotDefine() {
	fmtc.ErrorLn("The command is not defined. Run 'madock help' to invoke help")
}
