package db

import (
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
	"log"
	"strconv"
)

func Info() {
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
