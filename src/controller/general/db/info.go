package db

import (
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
	"strconv"
)

type ArgsInfoStruct struct {
	attr.Arguments
}

func Info() {
	getInfoArgs()

	projectConf := configs2.GetCurrentProjectConfig()
	if projectConf["platform"] != "pwa" {
		portsFile := paths.GetExecDirPath() + "/aruntime/ports.conf"
		portsConfig := configs2.ParseFile(portsFile)
		port, err := strconv.Atoi(portsConfig[configs2.GetProjectName()])
		if err != nil {
			log.Fatal(err)
		}
		fmtc.SuccessLn("host: db")
		fmtc.SuccessLn("name: " + projectConf["db/database"])
		fmtc.SuccessLn("user: " + projectConf["db/user"])
		fmtc.SuccessLn("password: " + projectConf["db/password"])
		fmtc.SuccessLn("root password: " + projectConf["db/root_password"])
		fmtc.SuccessLn("remote HOST:PORT: " + "localhost:" + strconv.Itoa(17000+((port-1)*20)+4))
	} else {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
	}
}

func getInfoArgs() *ArgsInfoStruct {
	args := new(ArgsInfoStruct)
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
