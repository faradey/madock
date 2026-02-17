package set

import (
	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
)

type ArgsStruct struct {
	attr.ArgumentsWithArgs
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"scope:set"},
		Handler:  Execute,
		Help:     "Set active scope",
		Category: "scope",
	})
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)
	scopes := configs.GetScopes(configs.GetProjectName())
	if len(args.Args) == 1 && len(scopes[args.Args[0]]) > 0 {
		result := configs.SetScope(configs.GetProjectName(), args.Args[0])
		if result {
			fmtc.SuccessLn("Scope was set successfully")
		} else {
			fmtc.WarningLn("Scope was not set")
		}
	} else {
		fmtc.WarningLn("There is no such scope")
	}
}
