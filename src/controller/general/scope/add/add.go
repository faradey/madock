package add

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
		Aliases:  []string{"scope:add"},
		Handler:  Execute,
		Help:     "Add scope",
		Category: "scope",
	})
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)
	scopes := configs.GetScopes(configs.GetProjectName())
	if len(args.Args) == 1 && len(scopes[args.Args[0]]) == 0 {
		result := configs.AddScope(configs.GetProjectName(), args.Args[0])
		if result {
			fmtc.SuccessLn("Scope was added and activated successfully")
		} else {
			fmtc.WarningLn("Scope was not added")
		}
	} else {
		fmtc.WarningLn("This scope is exist")
	}
}
