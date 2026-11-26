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
		Help:       "List docker volumes, networks and images, and projects/<name> directories, left by projects the registry no longer knows; --remove <name> takes one project's docker resources away. Supports --json (-j) output",
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
	// Unknown is each source that could not tell — docker's labels aside, a
	// directory that cannot be read or a zfs that does not answer. An empty
	// orphans list with something here is not "nothing left".
	Unknown []docker.OrphanUnknown `json:"unknown"`
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
// The list removes nothing. `--remove <name>` takes one project's leftovers
// away, behind `allow_destructive_commands` like `project:remove` — anything
// else would be a second route past that switch.
func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralProjectOrphans)).(*arg_struct.ControllerGeneralProjectOrphans)

	// Every name the registry holds counts as known, including entries it can no
	// longer read: a broken entry is somebody's to fix and belongs to
	// `project:list --stale`, not here.
	known := map[string]bool{}
	for _, entry := range configs.ListProjects() {
		known[entry.Name] = true
	}

	if args.Remove != "" {
		if args.Json {
			fmtc.ErrorLn("--json lists; it does not go with --remove")
			os.Exit(1)
		}
		remove(args.Remove, args.Force, known)
		return
	}
	if args.Force {
		fmtc.ErrorLn("--force goes with --remove <name>; the list on its own removes nothing")
		os.Exit(1)
	}

	found, unknown := docker.FindAllOrphans(known)
	if found == nil {
		found = []docker.Orphan{}
	}
	if unknown == nil {
		unknown = []docker.OrphanUnknown{}
	}

	if args.Json {
		output.PrintJSON(Output{Orphans: found, Unknown: unknown})
		return
	}

	for _, u := range unknown {
		fmtc.ErrorLn(fmt.Sprintf("Could not tell about %s: %s", u.Source, u.Error))
	}

	if len(found) == 0 {
		if len(unknown) > 0 {
			// "Nothing" would be an answer about what could not be read.
			os.Exit(1)
		}
		fmtc.SuccessLn("Nothing on this machine belongs to a project the registry has forgotten")
		return
	}

	current := ""
	listedOnly := false
	for _, orphan := range found {
		if orphan.Project != current {
			current = orphan.Project
			fmtc.WarningLn(fmt.Sprintf("%s — no entry in this installation", current))
		}
		fmt.Printf("  %-9s %s\n", orphan.Kind, orphan.Name)
		switch orphan.Kind {
		case "volume", "network", "image":
		default:
			listedOnly = true
		}
	}

	fmt.Println()
	fmtc.ToDoLn("Nothing was removed. To see what one project's removal would take: madock project:orphans --remove <name>")
	if listedOnly {
		fmtc.ToDoLn("Directories and datasets are listed only: --remove takes docker volumes, networks and images, never data outside docker.")
	}

	// Non-zero so a check can ask the question, the way project:list --stale
	// does. What is found is not an error, but it is something to act on.
	os.Exit(1)
}

// remove takes away what is left of one forgotten project, or, without force,
// says what that would be.
//
// One name per call, on purpose: a removal of every orphan at once would act
// on a list nobody read, and the list is exactly what gets things wrong — every
// fix in orphansOf so far was a live database it had listed.
func remove(project string, force bool, known map[string]bool) {
	// Before anything is read, as in project:remove. Without --force nothing is
	// removed either, but the plain list already answers that question on an
	// installation that may not act on it.
	if !configs.AllowsDestructiveCommands() {
		for _, line := range configs.DestructiveRefusal("project:orphans --remove") {
			fmtc.ErrorLn(line)
		}
		os.Exit(1)
	}

	if refusal := docker.OrphanRemovalRefusal(project, known); refusal != "" {
		fmtc.ErrorLn(refusal)
		os.Exit(1)
	}

	plan := docker.PlanOrphanRemoval(project, known)
	if plan.Empty() {
		fmtc.SuccessLn("Nothing labelled madock_" + project + " is left on this machine")
		return
	}

	if force {
		fmtc.WarningLn("Removing what is left of '" + project + "':")
	} else {
		fmtc.WarningLn("This would remove what is left of '" + project + "':")
	}
	for _, name := range plan.Volumes {
		fmt.Printf("  %-8s %s\n", "volume", name)
	}
	for _, name := range plan.Networks {
		fmt.Printf("  %-8s %s\n", "network", name)
	}
	for _, img := range plan.Images {
		fmt.Printf("  %-8s %s (%s)\n", "image", img.Name, img.ID)
	}

	if !force {
		fmt.Println()
		fmtc.ToDoLn("Nothing was removed. To remove these: madock project:orphans --remove " + project + " --force")
		return
	}

	removed, refused := docker.RemoveOrphans(plan)
	fmt.Println()
	if len(refused) > 0 {
		for _, line := range refused {
			fmtc.ErrorLn("  not removed: " + line)
		}
		fmtc.ErrorLn(fmt.Sprintf("Removed %d, docker refused %d", removed, len(refused)))
		os.Exit(1)
	}
	fmtc.SuccessLn(fmt.Sprintf("Removed %d", removed))
}
