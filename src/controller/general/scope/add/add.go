package add

import (
	"os"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
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
		ArgsType: new(ArgsStruct),
	})
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)
	scopes := configs.GetScopes(configs.GetProjectName())
	if len(args.Args) == 1 && len(scopes[args.Args[0]]) == 0 {
		// A scope is a second Compose project, <project>-<scope>. Refused when
		// another project already runs under that name: the scope's first start
		// would take over its containers and volumes.
		if clash := configs.ScopeComposeClash(configs.GetProjectName(), args.Args[0]); clash != "" {
			fmtc.ErrorLn(clash)
			os.Exit(1)
		}
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
