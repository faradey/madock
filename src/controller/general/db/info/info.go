package info

import (
	"fmt"
	"strconv"

	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/ports"
)

type ArgsStruct struct {
	attr.Arguments
}

func Info() {
	attr.Parse(new(ArgsStruct))

	projectConf := configs2.GetCurrentProjectConfig()
	projectName := configs2.GetProjectName()
	if projectConf["platform"] != "pwa" {
		fmtc.SuccessLn("First DB")
		fmtc.SuccessLn("   host: db")
		fmtc.SuccessLn("   name: " + projectConf["db/database"])
		fmtc.SuccessLn("   user: " + projectConf["db/user"])
		fmtc.SuccessLn("   password: " + projectConf["db/password"])
		fmtc.SuccessLn("   root password: " + projectConf["db/root_password"])
		fmtc.SuccessLn("   remote HOST:PORT: " + "localhost:" + strconv.Itoa(ports.GetPort(projectName, ports.ServiceDB)))

		fmt.Println("")
		fmtc.SuccessLn("Second DB")
		fmtc.SuccessLn("   host: db2")
		fmtc.SuccessLn("   name: " + projectConf["db2/database"])
		fmtc.SuccessLn("   user: " + projectConf["db2/user"])
		fmtc.SuccessLn("   password: " + projectConf["db2/password"])
		fmtc.SuccessLn("   root password: " + projectConf["db2/root_password"])
		fmtc.SuccessLn("   remote HOST:PORT: " + "localhost:" + strconv.Itoa(ports.GetPort(projectName, ports.ServiceDB2)))
	} else {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
	}
}
