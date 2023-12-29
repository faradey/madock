package restart

import (
	"github.com/faradey/madock/src/controller/general/start"
	"github.com/faradey/madock/src/controller/general/stop"
	"github.com/faradey/madock/src/helper/cli/attr"
)

type ArgsStruct struct {
	attr.Arguments
	WithChown bool `arg:"-c,--with-chown" help:"With Chown"`
}

func Execute() {
	attr.Parse(new(ArgsStruct))

	stop.Execute()
	start.Execute()
}
