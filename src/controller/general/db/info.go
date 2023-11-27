package db

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"strconv"
)

type ArgsInfoStruct struct {
	attr.Arguments
}

func Info() {
	getInfoArgs()

	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["PLATFORM"] != "pwa" {
		portsFile := paths.GetExecDirPath() + "/aruntime/ports.conf"
		portsConfig := configs.ParseFile(portsFile)
		port, err := strconv.Atoi(portsConfig[configs.GetProjectName()])
		if err != nil {
			log.Fatal(err)
		}
		fmtc.SuccessLn("host: db")
		fmtc.SuccessLn("name: " + projectConf["DB_DATABASE"])
		fmtc.SuccessLn("user: " + projectConf["DB_USER"])
		fmtc.SuccessLn("password: " + projectConf["DB_PASSWORD"])
		fmtc.SuccessLn("root password: " + projectConf["DB_ROOT_PASSWORD"])
		fmtc.SuccessLn("remote HOST:PORT: " + "localhost:" + strconv.Itoa(17000+((port-1)*20)+4))
	} else {
		fmtc.Warning("This command is not supported for " + projectConf["PLATFORM"])
	}
}

func getInfoArgs() *ArgsInfoStruct {
	args := new(ArgsInfoStruct)
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
