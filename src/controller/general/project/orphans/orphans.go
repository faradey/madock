package orphans

import (
	"fmt"
	"os"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/cli/output"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
)

func init() {
	command.Register(&command.Definition{
		Aliases:    []string{"project:orphans"},
		JSONOutput: true,
		Handler:    Execute,
		Help:       "List docker volumes, networks and images left by projects the registry no longer knows. Supports --json (-j) output",
		Category:   "project",
		// Global for the same reason project:list is: it describes the
		// installation rather than a project, and the reason to run it is
		// usually that a project is gone.
		Global: true,
	})
}

// Output is the --json payload.
type Output struct {
	Orphans []docker.Orphan `json:"orphans"`
}

// Execute prints what is left on the machine of projects that no longer exist.
//
// A project's volume outlives its directory, and after that nothing named it:
// it is not in the registry, `project:list` reads the registry, and
// `project:remove` needs a registry entry there is none of. On a server, where
// destructive commands are switched off as shipped, the only cleanup available
// was to delete the directories and leave the volumes — turning litter that can
// be seen into litter that cannot.
//
// **It removes nothing, and prints the command rather than running it.**
// Deleting these stays with `project:remove` behind
// `allow_destructive_commands`; a removal here would be a second route past
// that switch.
func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralProjectOrphans)).(*arg_struct.ControllerGeneralProjectOrphans)

	// Every name the registry holds counts as known, including entries it can no
	// longer read: a broken entry is somebody's to fix and belongs to
	// `project:list --stale`, not here.
	known := map[string]bool{}
	for _, entry := range configs.ListProjects() {
		known[entry.Name] = true
	}

	found := docker.FindOrphans(known)

	if args.Json {
		output.PrintJSON(Output{Orphans: found})
		return
	}

	if len(found) == 0 {
		fmtc.SuccessLn("Nothing on this machine belongs to a project the registry has forgotten")
		return
	}

	current := ""
	for _, orphan := range found {
		if orphan.Project != current {
			current = orphan.Project
			fmtc.WarningLn(fmt.Sprintf("%s — no entry in this installation", current))
		}
		fmt.Printf("  %-8s %s\n", orphan.Kind, orphan.Name)
	}

	fmt.Println()
	fmtc.ToDoLn("These are not removed here. Set up the project again and use project:remove, " +
		"or remove them by hand — docker volume rm <name>, docker network rm <name>, docker image rm <name>")

	// Non-zero so a check can ask the question, the way project:list --stale
	// does. What is found is not an error, but it is something to act on.
	os.Exit(1)
}
