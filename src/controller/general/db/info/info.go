package info

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/cli/output"
	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/dbtarget"
	"github.com/faradey/madock/v3/src/helper/ports"
)

type DbInfoOutput struct {
	Databases []DatabaseInfo `json:"databases"`
}

type DatabaseInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Host         string `json:"host"`
	Database     string `json:"database"`
	User         string `json:"user"`
	Password     string `json:"password"`
	RootPassword string `json:"root_password,omitempty"`
	RemoteHost   string `json:"remote_host"`
	RemotePort   int    `json:"remote_port"`
	// Shared and Provider describe a database owned by another project. They
	// stay absent for the ordinary case of a database inside this project.
	Shared   bool   `json:"shared,omitempty"`
	Provider string `json:"provider,omitempty"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"db:info"},
		Handler:  Info,
		Help:     "Show database info. Supports --json (-j) output",
		Category: "database",
		ArgsType: new(arg_struct.ControllerGeneralDbInfo),
	})
}

func Info() {
	args := attr.Parse(new(arg_struct.ControllerGeneralDbInfo)).(*arg_struct.ControllerGeneralDbInfo)

	projectConf := configs2.GetCurrentProjectConfig()
	projectName := configs2.GetProjectName()

	var databases []DatabaseInfo
	if db, ok := describe(projectConf, projectName, "db", "First DB"); ok {
		databases = append(databases, db)
	}
	if db2, ok := describe(projectConf, projectName, "db2", "Second DB"); ok {
		databases = append(databases, db2)
	}

	if args.Json {
		output.PrintJSON(DbInfoOutput{Databases: databases})
		return
	}

	if len(databases) == 0 {
		fmtc.WarningLn("This project has no database: db/enabled is false and no shared database is configured.")
		return
	}

	for i, db := range databases {
		if i > 0 {
			fmt.Println("")
		}
		title := db.Name
		if db.Shared {
			title += " (shared from " + db.Provider + ")"
		}
		fmtc.SuccessLn(title)
		fmtc.SuccessLn("   type: " + strings.ToUpper(db.Type))
		fmtc.SuccessLn("   host: " + db.Host)
		fmtc.SuccessLn("   name: " + db.Database)
		fmtc.SuccessLn("   user: " + db.User)
		fmtc.SuccessLn("   password: " + db.Password)
		if db.RootPassword != "" {
			fmtc.SuccessLn("   root password: " + db.RootPassword)
		}
		fmtc.SuccessLn("   remote HOST:PORT: " + db.RemoteHost + ":" + strconv.Itoa(db.RemotePort))
	}
}

// describe builds the info row for one service, or reports false when the
// project neither runs that service nor borrows it from another project.
func describe(projectConf map[string]string, projectName, service, label string) (DatabaseInfo, bool) {
	target, ok := dbtarget.Resolve(projectConf, projectName, service)
	if !ok {
		return DatabaseInfo{}, false
	}

	info := DatabaseInfo{
		Name:     label,
		Type:     target.Type,
		Host:     target.Host,
		Database: target.Database,
		User:     target.User,
		Password: target.Password,
		// The published port belongs to the project running the container, which
		// is the provider when the database is shared.
		RemoteHost: "localhost",
		RemotePort: ports.GetPort(target.Project, portService(target.Service)),
	}
	// root_password is only meaningful for MySQL/MariaDB.
	if target.Type == "mysql" {
		info.RootPassword = target.RootPassword
	}
	if target.Shared {
		info.Shared = true
		info.Provider = target.Project
		// A shared server is addressed by container name over the proxy network,
		// not by the compose service alias of this project.
		info.Host = target.Container
	}

	return info, true
}

func portService(service string) string {
	if service == "db2" {
		return ports.ServiceDB2
	}
	return ports.ServiceDB
}
