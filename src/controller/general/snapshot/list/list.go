package list

import (
	"fmt"
	"path/filepath"
	"sort"

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
	out := ListJSON{Snapshots: Read(dir)}

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

// Read lists the snapshots in dir, oldest first.
//
// By the timestamp at the end of the name, not by the name: `snapshot:create
// -n zzz` sorts after a newer `-n aaa` as a string.
func Read(dir string) []create.SnapshotJSON {
	snapshots := []create.SnapshotJSON{}
	if !paths.IsFileExist(dir) {
		return snapshots
	}
	names := paths.GetDirs(dir)
	sort.SliceStable(names, func(i, j int) bool {
		a, b := stamp(filepath.Base(names[i])), stamp(filepath.Base(names[j]))
		if a != b {
			return a < b
		}
		return names[i] < names[j]
	})
	for _, name := range names {
		snapshots = append(snapshots, create.Describe(dir, filepath.Base(name)))
	}
	return snapshots
}

// stamp is the "2006-01-02-15-04-05" suffix snapshot:create writes.
func stamp(name string) string {
	const width = len("2006-01-02-15-04-05")
	if len(name) < width {
		return name
	}
	return name[len(name)-width:]
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
