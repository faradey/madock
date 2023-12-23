package restart

import (
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/controller/general/start"
	"github.com/faradey/madock/src/controller/general/stop"
	"github.com/faradey/madock/src/helper/cli/attr"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
	WithChown bool `arg:"-c,--with-chown" help:"With Chown"`
}

func Execute() {
	getArgs()

	stop.Execute()
	start.Execute()
}

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if attr.IsParseArgs && len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		p, err := arg.NewParser(arg.Config{
			IgnoreEnv: true,
		}, args)

		if err != nil {
			log.Fatal(err)
		}

		err = p.Parse(argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	attr.IsParseArgs = false
	return args
}
