package proxy

import (
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
	Force bool `arg:"-f,--force" help:"Force"`
}

func Execute(flag string) {
	args := getArgs()

	if !configs2.IsHasNotConfig() {
		projectConf := configs2.GetCurrentProjectConfig()
		if projectConf["PROXY_ENABLED"] == "true" {
			if flag == "prune" {
				docker.DownNginx(args.Force)
			} else if flag == "stop" {
				docker.StopNginx(args.Force)
			} else if flag == "restart" {
				docker.StopNginx(args.Force)
				docker.UpNginx()
			} else if flag == "start" {
				docker.UpNginx()
			} else if flag == "rebuild" {
				docker.DownNginx(args.Force)
				docker.UpNginx()
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
	if attr.IsParseArgs && len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		p, err := arg.NewParser(arg.Config{
			IgnoreEnv: true,
		}, args)

		if err != nil {
			log.Fatal(err)
		}

		err = p.Parse(argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	attr.IsParseArgs = false
	return args
}