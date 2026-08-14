package list

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
)

type ArgsStruct struct {
	attr.Arguments
	Stale bool `arg:"--stale" help:"Only the entries whose source directory is gone"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"project:list"},
		Handler:  Execute,
		Help:     "List registered projects. --stale for the ones whose source is gone",
		Category: "project",
		ArgsType: new(ArgsStruct),
		// Global: it describes the installation, not a project, and the reason to
		// run it is usually that something about the registry is wrong — which is
		// not a moment to require standing in a working project.
		Global: true,
	})
}

// Execute prints the registry and what is true about each entry.
//
// It exists for two reasons. Three commands in madock-pro tell the user to "run
// madock project:list" when a provider name is not found, and until now that was
// a command that did not exist. And the registry drifts: an entry whose source
// directory has been deleted stays, keeps its port reservations, and keeps a
// server block in the shared proxy routing its hosts at containers that cannot
// exist — on the machine this was written on, two of fifty-eight entries were in
// that state and both were still in the generated proxy configuration.
//
// Nothing here changes anything. Removing an entry is project:remove, run from the
// project's own directory, which refuses when that directory is not the one
// recorded.
func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

	entries := configs.ListProjects()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	stale := make([]configs.ProjectEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.State != configs.ProjectOk {
			stale = append(stale, entry)
		}
	}

	shown := entries
	if args.Stale {
		shown = stale
	}

	if args.Json {
		out, err := json.MarshalIndent(shown, "", "  ")
		if err != nil {
			logger.Fatal(err)
		}
		fmt.Println(string(out))
		return
	}

	if len(shown) == 0 {
		if args.Stale {
			fmtc.SuccessLn("Every registered project still has its source directory")
			return
		}
		fmtc.WarningLn("No projects are registered in this installation")
		return
	}

	for _, entry := range shown {
		switch entry.State {
		case configs.ProjectMissingSource:
			fmtc.ErrorLn(fmt.Sprintf("%-28s source is gone: %s", entry.Name, entry.Path))
		case configs.ProjectNoPath:
			fmtc.WarningLn(fmt.Sprintf("%-28s no path recorded", entry.Name))
		default:
			fmt.Printf("%-28s %s\n", entry.Name, entry.Path)
		}
	}

	// Said once, at the end, and only when there is something to say: the point of
	// the summary is that a stale entry is easy to miss in a list of fifty.
	if !args.Stale && len(stale) > 0 {
		fmt.Println()
		fmtc.WarningLn(fmt.Sprintf("%d of %d entries have no source directory. They keep their ports and their routing.", len(stale), len(entries)))
		fmtc.ToDoLn("Run madock project:list --stale to see only those")
	}

	if args.Stale {
		// A non-zero exit so a script can ask the question, but only for the
		// filtered form: the plain list answering non-zero would be absurd.
		os.Exit(1)
	}
}
