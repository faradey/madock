package info

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/cli/output"
	configs2 "github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/dbtarget"
	"github.com/faradey/madock/v4/src/helper/ports"
)

type DbInfoOutput struct {
	Databases []DatabaseInfo `json:"databases"`
}

type DatabaseInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Database string `json:"database"`
	User     string `json:"user"`
	// The passwords are present unless the edition withholds them, in which
	// case they arrive only when asked for by name with --show-secrets. The
	// booleans are always there, so a script can tell "no password" from "not
	// shown" without the value.
	Password        string `json:"password,omitempty"`
	PasswordSet     bool   `json:"password_set"`
	RootPassword    string `json:"root_password,omitempty"`
	RootPasswordSet bool   `json:"root_password_set,omitempty"`
	RemoteHost      string `json:"remote_host"`
	RemotePort      int    `json:"remote_port"`
	// Shared and Provider describe a database owned by another project. They
	// stay absent for the ordinary case of a database inside this project.
	Shared   bool   `json:"shared,omitempty"`
	Provider string `json:"provider,omitempty"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:    []string{"db:info"},
		JSONOutput: true,
		Handler:    Info,
		Help:       "Show database info. Supports --json (-j) output",
		Category:   "database",
		ArgsType:   new(arg_struct.ControllerGeneralDbInfo),
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
		// The values go, the two booleans stay: a script can still tell a
		// database with no password from one whose password was not printed,
		// which is the only thing it could honestly do with a masked string.
		// JSON is not the safer half — it is what ends up in a CI log.
		//
		// Only where the edition withholds secrets. In the open-source edition
		// the fields are present as they always were, so a script reading a
		// password out of `db:info --json` on a developer's laptop keeps
		// working without learning a new flag.
		if fmtc.HideSecretsByDefault && !args.ShowSecrets {
			for i := range databases {
				databases[i].Password = ""
				databases[i].RootPassword = ""
			}
		}
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
		fmtc.SuccessLn("   password: " + fmtc.SecretOrValue(db.Password, args.ShowSecrets))
		if db.RootPasswordSet {
			fmtc.SuccessLn("   root password: " + fmtc.SecretOrValue(db.RootPassword, args.ShowSecrets))
		}
		if db.RemotePort > 0 {
			fmtc.SuccessLn("   remote HOST:PORT: " + db.RemoteHost + ":" + strconv.Itoa(db.RemotePort))
		} else {
			// Printing localhost:0 would read as a port rather than as its
			// absence, and the reason it is absent is worth saying.
			fmtc.SuccessLn("   remote HOST:PORT: not published yet — run madock start")
		}
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
		Name:        label,
		Type:        target.Type,
		Host:        target.Host,
		Database:    target.Database,
		User:        target.User,
		Password:    target.Password,
		PasswordSet: target.Password != "",
		// The published port belongs to the project running the container, which
		// is the provider when the database is shared.
		//
		// Looked up, never allocated: this command answers a question and must
		// not change the thing it is describing. `madock info` already carries
		// that rule; this one used to call GetOrAllocate, so asking about a
		// project that had never started reserved ports for every service it
		// might one day run, and then printed one nothing was listening on.
		// Zero means "not allocated yet", which is the honest answer before the
		// first start.
		RemoteHost: "localhost",
		RemotePort: ports.GetRegistry().Get(target.Project, portService(target.Service)),
	}
	// root_password is only meaningful for MySQL/MariaDB.
	//
	// It is also the most valuable line this command prints, and the reason the
	// default is now to describe it rather than show it: on a project that
	// borrows a shared database, this is the **provider's** root password, which
	// reaches every other project's schema on that server. The consumer's own
	// account cannot.
	if target.Type == "mysql" {
		info.RootPassword = target.RootPassword
		info.RootPasswordSet = target.RootPassword != ""
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
