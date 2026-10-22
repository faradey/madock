package list

import (
	"fmt"
	"github.com/faradey/madock/v4/src/helper/snapshotengine"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/general/snapshot/create"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/output"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/paths"
)

// snapshot:restore --name needs a name, and the only list of names there was
// is the one restore prints before it asks for a number — which a program
// cannot answer. This is that list on its own.

type ArgsStruct struct {
	attr.Arguments
}

// ListJSON is the --json answer of snapshot:list.
type ListJSON struct {
	Snapshots []create.SnapshotJSON `json:"snapshots"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:    []string{"snapshot:list"},
		Handler:    Execute,
		Help:       "List this project's snapshots, oldest first. Supports --json (-j)",
		Category:   "snapshot",
		ArgsType:   new(ArgsStruct),
		JSONOutput: true,
	})
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

	dir := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/backup/snapshot"
	project := snapshotengine.Project{Name: configs.GetProjectName(), Conf: configs.GetCurrentProjectConfig()}
	out := ListJSON{Snapshots: ReadFor(project, dir)}

	if args.Json {
		output.Begin().Print(out)
		return
	}

	if len(out.Snapshots) == 0 {
		fmt.Println("No snapshots. Create one with 'madock snapshot:create'.")
		return
	}
	for _, s := range out.Snapshots {
		fmt.Printf("  %-45s %8s  %v\n", s.Name, humanBytes(s.SizeBytes), s.Files)
	}
}

// ReadFor is Read with each engine snapshot's own size (engine_bytes).
func ReadFor(p snapshotengine.Project, dir string) []create.SnapshotJSON {
	snapshots := []create.SnapshotJSON{}
	for _, name := range create.OrderedNames(dir) {
		snapshots = append(snapshots, create.DescribeFor(p, dir, name))
	}
	return snapshots
}

// Read lists the snapshots in dir, oldest first (create.OrderedNames).
func Read(dir string) []create.SnapshotJSON {
	snapshots := []create.SnapshotJSON{}
	for _, name := range create.OrderedNames(dir) {
		snapshots = append(snapshots, create.Describe(dir, name))
	}
	return snapshots
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
