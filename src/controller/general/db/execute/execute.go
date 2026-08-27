package execute

import (
	"os"
	"strconv"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/cli/shell"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/dbtarget"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:    []string{"db:execute", "db:e"},
		JSONOutput: true,
		Handler:    Execute,
		Help:       "Execute SQL query. Supports --json (-j) output on MySQL, MariaDB and PostgreSQL",
		Category:   "database",
		ArgsType:   new(arg_struct.ControllerGeneralDbExecute),
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

	clientArgs := []string{target.MySQLClient(), "-u", login, "-p" + loginPassword, "-h", target.Host}
	if args.Json {
		clientArgs = append(clientArgs, "--xml")
	}
	clientArgs = append(clientArgs, target.Database, "-e", args.Query)

	cmd, err := docker.PrepareContainerExec(target.Container, user, false, clientArgs...)
	if err != nil {
		logger.FatalChild(err)
	}
	cmd.Stderr = os.Stderr

	if !args.Json {
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			logger.Fatal(err)
		}

		return
	}

	// Streamed rather than buffered: this is the command a table dump is taken
	// with, and holding the whole result in memory twice is a cost for the case
	// that matters most.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		logger.Fatal(err)
	}

	sets, convertErr := mysqlXMLToJSON(stdout, os.Stdout)

	// Waited on before the conversion error is reported: the client's own message
	// on stderr says more about a broken query than "unexpected EOF" does.
	if err := cmd.Wait(); err != nil {
		logger.Fatal(err)
	}
	if convertErr != nil {
		logger.Fatal(convertErr)
	}

	// Several statements in one -e produce several result sets, and they are
	// concatenated into the one array. Said out loud, because a consumer counting
	// rows would otherwise have no way to know two queries were merged.
	if sets > 1 {
		fmtc.WarningLn("The query returned " + strconv.Itoa(sets) + " result sets; their rows are concatenated into one JSON array")
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
	query := args.Query
	format := ""
	if args.Json {
		// PostgreSQL encodes the rows itself, so nothing here has to know the
		// columns or undo an escaping. -A -t strips psql's alignment and its
		// header, which would otherwise be printed around the value.
		query = postgresJSONQuery(query)
		format = " -A -t"
	}

	command := "PGPASSWORD=" + shell.Quote(target.Password) +
		" psql -U " + shell.Quote(target.User) +
		" -h " + shell.Quote(target.Host) +
		" " + shell.Quote(target.Database) +
		format +
		" -c " + shell.Quote(query)

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

	// Refused rather than approximated, which is the whole lesson of this change.
	//
	// The other two engines are asked for a format they produce themselves. Here
	// there is nothing to ask: the argument is JavaScript evaluated by mongosh,
	// and turning it into JSON means wrapping it — `EJSON.stringify(<eval>)` —
	// which is a guess about what the expression returns. A cursor needs
	// `.toArray()` and a statement returns nothing at all, so the wrap would
	// change the meaning of some queries and silently empty others. An honest
	// refusal costs a person one line of documentation; a wrapper that is right
	// most of the time costs whatever was in the file nobody re-read.
	if args.Json {
		fmtc.ErrorLn("db:execute --json is not supported for MongoDB")
		fmtc.ToDoLn("mongosh prints structured output already — shape it in the query itself, e.g. EJSON.stringify(db.collection.find().toArray())")
		os.Exit(1)
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
