package execute

import (
	"os"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
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

	projectName := configs.GetProjectName()
	containerName := docker.GetContainerName(projectConf, projectName, service)

	dbType := configs.GetDbType(projectConf)

	switch dbType {
	case "postgresql":
		executePostgresql(containerName, projectConf, args, service)
	case "mongodb":
		executeMongodb(containerName, projectConf, args)
	default:
		executeMysql(containerName, projectConf, args, service)
	}
}

func executeMysql(containerName string, projectConf map[string]string, args *arg_struct.ControllerGeneralDbExecute, service string) {
	user := "mysql"
	if args.User != "" {
		user = args.User
	}

	mysqlCommandName := "mysql"
	if projectConf["db/repository"] == "mariadb" && configs.CompareVersions(projectConf["db/version"], "10.5") != -1 {
		mysqlCommandName = "mariadb"
	}

	cmd, err := docker.PrepareContainerExec(containerName, user, false, mysqlCommandName, "-u", "root", "-p"+projectConf["db/root_password"], "-h", service, projectConf["db/database"], "-e", args.Query)
	if err != nil {
		logger.Fatal(err)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Fatal(err)
	}
}

func executePostgresql(containerName string, projectConf map[string]string, args *arg_struct.ControllerGeneralDbExecute, service string) {
	user := "postgres"
	if args.User != "" {
		user = args.User
	}

	cmd, err := docker.PrepareContainerExec(containerName, user, false, "psql", "-U", projectConf["db/user"], "-h", service, projectConf["db/database"], "-c", args.Query)
	if err != nil {
		logger.Fatal(err)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Fatal(err)
	}
}

func executeMongodb(containerName string, projectConf map[string]string, args *arg_struct.ControllerGeneralDbExecute) {
	user := "root"
	if args.User != "" {
		user = args.User
	}

	cmd, err := docker.PrepareContainerExec(containerName, user, false, "mongosh", "--username="+projectConf["db/user"], "--password="+projectConf["db/password"], "--authenticationDatabase=admin", projectConf["db/database"], "--eval", args.Query)
	if err != nil {
		logger.Fatal(err)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Fatal(err)
	}
}
