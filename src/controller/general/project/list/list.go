package list

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
)

type ArgsStruct struct {
	attr.Arguments
	Stale   bool `arg:"--stale" help:"Only the entries whose source directory is gone"`
	Running bool `arg:"--running" help:"Only the projects that have containers running"`
}

// projectRow is a registry entry plus what is true of it right now.
//
// Running is a pointer so that "not running" and "docker could not be asked" are
// different values in JSON — false and null. They are different facts, and a
// reader who cannot tell them apart will read an unanswered question as an
// answer. The registry fields stay in configs.ProjectEntry, where they belong:
// which projects exist is a property of the installation, and which are up is
// not.
type projectRow struct {
	configs.ProjectEntry
	Running *bool `json:"running"`
}

// withRunning pairs each registry entry with whether it is up.
//
// `known` false means docker could not be asked, and then every Running stays
// nil — which is null in JSON and a blank column in the text. That distinction
// is the whole reason this is a pointer: "no projects are running" and "nobody
// could find out" are different facts, and reporting the first when the second
// is true is a confident wrong answer.
func withRunning(entries []configs.ProjectEntry, active []string, known bool) []projectRow {
	running := make(map[string]bool, len(active))
	for _, name := range active {
		running[name] = true
	}

	rows := make([]projectRow, 0, len(entries))
	for _, entry := range entries {
		row := projectRow{ProjectEntry: entry}
		if known {
			// A fresh variable per row: one shared bool would give every row a
			// pointer to the last answer.
			isRunning := running[entry.Name]
			row.Running = &isRunning
		}
		rows = append(rows, row)
	}

	return rows
}

func init() {
	command.Register(&command.Definition{
		Aliases:    []string{"project:list"},
		JSONOutput: true,
		Handler:    Execute,
		Help:       "List registered projects. --running for the ones that are up, --stale for the ones whose source is gone. Supports --json (-j) output",
		Category:   "project",
		ArgsType:   new(ArgsStruct),
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

	// Asked once for the whole registry — one `docker ps`, not a status call per
	// project. The question is "who is eating memory", and answering it used to
	// mean walking the projects and running `status` in each.
	active, activeErr := paths.ActiveProjects()
	rows := withRunning(shown, active, activeErr == nil)

	// --running on an answer nobody has is a refusal, not an empty list. The
	// filtered form exists to be acted on, and "nothing is running" would be a
	// lie dressed as a result.
	if args.Running {
		if activeErr != nil {
			fmtc.ErrorLn("Cannot tell which projects are running: " + activeErr.Error())
			os.Exit(1)
		}
		filtered := rows[:0]
		for _, row := range rows {
			if row.Running != nil && *row.Running {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}

	if args.Json {
		out, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			logger.Fatal(err)
		}
		fmt.Println(string(out))
		return
	}

	// Said before the list, because every line below is missing a column and the
	// reader has to know that before reading them rather than after.
	if activeErr != nil {
		fmtc.WarningLn("Docker could not be asked which projects are running: " + activeErr.Error())
	}

	if len(rows) == 0 {
		if args.Running {
			fmtc.WarningLn("No registered project has containers running")
			return
		}
		if args.Stale {
			fmtc.SuccessLn("Every registry entry resolves, and every project still has its source directory")
			return
		}
		fmtc.WarningLn("No projects are registered in this installation")
		return
	}

	for _, row := range rows {
		// Blank rather than "stopped" when the answer is unknown, and blank
		// rather than a word for a stopped project: the column exists to make a
		// running project findable in a list of fifty, and filling it in for
		// every line defeats that.
		state := "        "
		if row.Running != nil && *row.Running {
			state = "running "
		}

		switch row.State {
		case configs.ProjectMissingSource:
			fmtc.ErrorLn(fmt.Sprintf("%-28s %ssource is gone: %s", row.Name, state, row.Path))
		case configs.ProjectBrokenLink:
			fmtc.ErrorLn(fmt.Sprintf("%-28s %sregistry entry links to nothing: %s", row.Name, state, row.Path))
		case configs.ProjectNoPath:
			fmtc.WarningLn(fmt.Sprintf("%-28s %sno path recorded", row.Name, state))
		default:
			fmt.Printf("%-28s %s%s\n", row.Name, state, row.Path)
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
