package restart

import (
	"os"
	"strings"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/general/start"
	"github.com/faradey/madock/v4/src/controller/general/stop"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/configs/aruntime/project"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"restart"},
		Handler:  Execute,
		Help:     "Restart containers",
		Category: "general",
		ArgsType: new(arg_struct.ControllerGeneralStart),
	})
}

// Seams for the ordering test. Nothing else replaces them.
var (
	parseArgs = func() *arg_struct.ControllerGeneralStart {
		return attr.Parse(new(arg_struct.ControllerGeneralStart)).(*arg_struct.ControllerGeneralStart)
	}
	checkTemplates  = project.ReportBrokenIncludes
	stopContainers  = stop.Execute
	startContainers = start.ExecuteWith
)

// Execute stops the project's containers and starts them again.
//
// The arguments are read first, and that order is the whole point of this
// function rather than an implementation detail.
//
// `restart` used to be stop-then-start with no parsing of its own; the parsing
// lived inside start, which is to say it happened after everything was already
// down. An argument this command does not take — `madock restart php`, meant as
// "restart just the php service" — therefore reached go-arg with every container
// stopped, and go-arg ends the process on a bad argument. The message read
// "too many positional arguments at 'php'", which sounds like nothing happened;
// what had actually happened was the whole environment going down and staying
// down. Measured on a production machine on 2026-08-18: nginx, php, db, redisdb
// and deployer stopped, the site off the air until somebody ran `madock start`.
//
// Parsing first turns that into a refusal that costs nothing.
func Execute() {
	// A bare word here is somebody asking for one service back, and there is a
	// command for that now. go-arg would refuse it too — that is the 3.9.16 fix
	// and it still holds — but "too many positional arguments at 'php'" is a
	// parser complaint that names neither what was wanted nor what does it.
	if len(os.Args) > 2 {
		if name := unexpectedPositional(os.Args[2:]); name != "" {
			fmtc.ErrorLn("`restart` restarts the whole project and takes no service name.")
			fmtc.WarningLn("To restart one service: madock service:restart " + name)
			os.Exit(1)
		}
	}

	args := parseArgs()

	// The same rule one step further out. `start` generates the build context,
	// which is after `stop` has run — so a template whose includes no longer
	// resolve took the environment down and left it down, exactly as a bad
	// argument used to. Measured on the demo machine on 2026-08-18, on a project
	// whose override still included a snippet that had moved.
	if checkTemplates(configs.GetProjectName()) {
		os.Exit(1)
	}

	stopContainers()
	startContainers(args)
}

// unexpectedPositional returns the first argument that is not a flag.
//
// Every flag this command takes is a boolean, so a bare word is never a flag's
// value and can be named without ambiguity. If that stops being true — a flag
// here that takes a value — this reads that value as a service name, and the
// check has to learn the flag.
func unexpectedPositional(argv []string) string {
	for _, arg := range argv {
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		return arg
	}
	return ""
}
