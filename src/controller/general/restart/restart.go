package restart

import (
	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/general/start"
	"github.com/faradey/madock/v3/src/controller/general/stop"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
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
	args := parseArgs()

	stopContainers()
	startContainers(args)
}
