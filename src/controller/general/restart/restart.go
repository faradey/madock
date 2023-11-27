package restart

import (
	"github.com/faradey/madock/src/controller/general/start"
	"github.com/faradey/madock/src/controller/general/stop"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
	WithChown bool `long:"with-chown" short:"c" description:"With Chown"`
}

func Execute() {
	getArgs()

	stop.Execute()
	start.Execute()
}

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		var err error
		_, err = flags.ParseArgs(args, argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	return args
}
