package info

import (
	"fmt"
	"strconv"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/cli/output"
	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/ports"
)

type DbInfoOutput struct {
	Databases []DatabaseInfo `json:"databases"`
}

type DatabaseInfo struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Database     string `json:"database"`
	User         string `json:"user"`
	Password     string `json:"password"`
	RootPassword string `json:"root_password"`
	RemoteHost   string `json:"remote_host"`
	RemotePort   int    `json:"remote_port"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"db:info"},
		Handler:  Info,
		Help:     "Show database info. Supports --json (-j) output",
		Category: "database",
	})
}

func Info() {
	args := attr.Parse(new(arg_struct.ControllerGeneralDbInfo)).(*arg_struct.ControllerGeneralDbInfo)

	projectConf := configs2.GetCurrentProjectConfig()
	projectName := configs2.GetProjectName()

	db1Port := ports.GetPort(projectName, ports.ServiceDB)
	db2Port := ports.GetPort(projectName, ports.ServiceDB2)

	databases := []DatabaseInfo{
		{
			Name:         "First DB",
			Host:         "db",
			Database:     projectConf["db/database"],
			User:         projectConf["db/user"],
			Password:     projectConf["db/password"],
			RootPassword: projectConf["db/root_password"],
			RemoteHost:   "localhost",
			RemotePort:   db1Port,
		},
		{
			Name:         "Second DB",
			Host:         "db2",
			Database:     projectConf["db2/database"],
			User:         projectConf["db2/user"],
			Password:     projectConf["db2/password"],
			RootPassword: projectConf["db2/root_password"],
			RemoteHost:   "localhost",
			RemotePort:   db2Port,
		},
	}

	if args.Json {
		output.PrintJSON(DbInfoOutput{Databases: databases})
		return
	}

	// Text output
	fmtc.SuccessLn("First DB")
	fmtc.SuccessLn("   host: db")
	fmtc.SuccessLn("   name: " + projectConf["db/database"])
	fmtc.SuccessLn("   user: " + projectConf["db/user"])
	fmtc.SuccessLn("   password: " + projectConf["db/password"])
	fmtc.SuccessLn("   root password: " + projectConf["db/root_password"])
	fmtc.SuccessLn("   remote HOST:PORT: " + "localhost:" + strconv.Itoa(db1Port))

	fmt.Println("")
	fmtc.SuccessLn("Second DB")
	fmtc.SuccessLn("   host: db2")
	fmtc.SuccessLn("   name: " + projectConf["db2/database"])
	fmtc.SuccessLn("   user: " + projectConf["db2/user"])
	fmtc.SuccessLn("   password: " + projectConf["db2/password"])
	fmtc.SuccessLn("   root password: " + projectConf["db2/root_password"])
	fmtc.SuccessLn("   remote HOST:PORT: " + "localhost:" + strconv.Itoa(db2Port))
}
