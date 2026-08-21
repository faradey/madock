package execute

import (
	"os"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/shell"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/dbtarget"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"db:execute", "db:e"},
		Handler:  Execute,
		Help:     "Execute SQL query",
		Category: "database",
		ArgsType: new(arg_struct.ControllerGeneralDbExecute),
	})
}

func Execute() {
	projectConf := configs.GetCurrentProjectConfig()
	args := attr.Parse(new(arg_struct.ControllerGeneralDbExecute)).(*arg_struct.ControllerGeneralDbExecute)

	service := "db"
	if args.DBServiceName != "" {
		service = args.DBServiceName
	}

	target := dbtarget.MustResolve(projectConf, configs.GetProjectName(), service)

	switch target.Type {
	case "postgresql":
		executePostgresql(target, args)
	case "mongodb":
		executeMongodb(target, args)
	default:
		executeMysql(target, args)
	}
}

func executeMysql(target dbtarget.Target, args *arg_struct.ControllerGeneralDbExecute) {
	user := "mysql"
	if args.User != "" {
		user = args.User
	}

	login, loginPassword := target.Login()
	cmd, err := docker.PrepareContainerExec(target.Container, user, false, target.MySQLClient(), "-u", login, "-p"+loginPassword, "-h", target.Host, target.Database, "-e", args.Query)
	if err != nil {
		logger.FatalChild(err)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Fatal(err)
	}
}

func executePostgresql(target dbtarget.Target, args *arg_struct.ControllerGeneralDbExecute) {
	user := "postgres"
	if args.User != "" {
		user = args.User
	}

	// psql takes no password on the command line, so it has to come from the
	// environment — which means a shell, which means every value in the line
	// has to be quoted. Without PGPASSWORD the command failed outright with
	// "fe_sendauth: no password supplied", so db:execute has never worked
	// against PostgreSQL. The query is quoted for the same reason and more
	// urgently: a WHERE clause containing an apostrophe is ordinary SQL.
	command := "PGPASSWORD=" + shell.Quote(target.Password) +
		" psql -U " + shell.Quote(target.User) +
		" -h " + shell.Quote(target.Host) +
		" " + shell.Quote(target.Database) +
		" -c " + shell.Quote(args.Query)

	cmd, err := docker.PrepareContainerExec(target.Container, user, false, "bash", "-c", command)
	if err != nil {
		logger.FatalChild(err)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Fatal(err)
	}
}

func executeMongodb(target dbtarget.Target, args *arg_struct.ControllerGeneralDbExecute) {
	user := "root"
	if args.User != "" {
		user = args.User
	}

	cmd, err := docker.PrepareContainerExec(target.Container, user, false, "mongosh", "--username="+target.User, "--password="+target.Password, "--authenticationDatabase=admin", target.Database, "--eval", args.Query)
	if err != nil {
		logger.FatalChild(err)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Fatal(err)
	}
}
