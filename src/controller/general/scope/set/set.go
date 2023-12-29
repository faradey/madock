package set

import (
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
)

type ArgsStruct struct {
	attr.ArgumentsWithArgs
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)
	scopes := configs.GetScopes(configs.GetProjectName())
	if len(args.Args) == 1 && len(scopes[args.Args[0]]) > 0 {
		result := configs.SetScope(configs.GetProjectName(), args.Args[0])
		if result {
			fmtc.SuccessLn("Scope was set")
		} else {
			fmtc.WarningLn("Scope was not set")
		}
	} else {
		fmtc.WarningLn("There is no such scope")
	}
}
