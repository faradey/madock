package version

import (
	"fmt"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/output"
	appversion "github.com/faradey/madock/v4/src/version"
)

func init() {
	command.Register(&command.Definition{
		// --version and -v are registered as command names, not flags: the
		// dispatcher matches os.Args[1] literally, so this is what makes
		// "madock --version" work at all.
		Aliases:    []string{"version", "--version", "-v"},
		JSONOutput: true,
		Handler:    Execute,
		Help:       "Show madock version. Supports --json (-j) output",
		Category:   "general",
		ArgsType:   new(arg_struct.ControllerGeneralVersion),
		// Global: answers about the binary, not a project.
		Global: true,
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralVersion)).(*arg_struct.ControllerGeneralVersion)

	if args.Json {
		output.PrintJSON(VersionOutput{Version: appversion.Version})
		return
	}

	fmt.Println("madock " + appversion.Version)
}

type VersionOutput struct {
	Version string `json:"version"`
}
