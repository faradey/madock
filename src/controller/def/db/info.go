package db

import (
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
	"log"
	"strconv"
)

func DBInfo() {
	projectConfig := configs.GetCurrentProjectConfig()
	if projectConfig["PLATFORM"] != "pwa" {
		portsFile := paths.GetExecDirPath() + "/aruntime/ports.conf"
		portsConfig := configs.ParseFile(portsFile)
		port, err := strconv.Atoi(portsConfig[configs.GetProjectName()])
		if err != nil {
			log.Fatal(err)
		}
		fmtc.SuccessLn("host: db")
		fmtc.SuccessLn("name: " + projectConfig["DB_DATABASE"])
		fmtc.SuccessLn("user: " + projectConfig["DB_USER"])
		fmtc.SuccessLn("password: " + projectConfig["DB_PASSWORD"])
		fmtc.SuccessLn("root password: " + projectConfig["DB_ROOT_PASSWORD"])
		fmtc.SuccessLn("remote HOST:PORT: " + "localhost:" + strconv.Itoa(17000+((port-1)*20)+4))
	} else {
		fmtc.Warning("This command is not supported for " + projectConfig["PLATFORM"])
	}
}
