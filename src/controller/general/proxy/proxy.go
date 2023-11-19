package proxy

import (
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
}

func Execute(flag string) {
	getArgs()

	if !configs.IsHasNotConfig() {
		projectConf := configs.GetCurrentProjectConfig()
		if projectConf["PROXY_ENABLED"] == "true" {
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
				builder.UpNginx()
			}
			fmtc.SuccessLn("Done")
		} else {
			fmtc.WarningLn("Proxy service is disabled. Run 'madock service:enable proxy' to enable it")
		}
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		var err error
		_, err = flags.ParseArgs(args, argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	return args
}
