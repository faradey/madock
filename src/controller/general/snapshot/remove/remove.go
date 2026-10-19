package remove

import (
	"fmt"
	"os"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/general/snapshot/create"
	"github.com/faradey/madock/v4/src/controller/general/snapshot/restore"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/cli/output"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/paths"
	"github.com/faradey/madock/v4/src/helper/snapshotengine"
)

// A snapshot could be made and restored and never removed. With tar that was a
// directory to delete by hand; with an engine it is not — the directory holds a
// marker, the data is a zfs or btrfs snapshot, and deleting the directory left
// that behind for good. Removal goes through create.Remove, which drops the
// engine's snapshot first, and snapshot/keep uses the same path.

type ArgsStruct struct {
	attr.Arguments
	Name string `arg:"-n,--name,required" help:"Snapshot to remove: its directory name, or the name given at creation (the newest one carrying it)"`
}

// RemovedJSON is the --json answer of snapshot:remove.
type RemovedJSON struct {
	Name string `json:"name"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:    []string{"snapshot:remove"},
		Handler:    Execute,
		Help:       "Remove a snapshot, the engine's copy included. Supports --json (-j)",
		Category:   "snapshot",
		ArgsType:   new(ArgsStruct),
		JSONOutput: true,
	})
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)
	var w *output.Writer
	if args.Json {
		w = output.Begin()
	}
	fail := func(msg string) {
		if w != nil {
			w.Fail("%s", msg)
		}
		fmtc.ErrorLn(msg)
		os.Exit(1)
	}

	project := snapshotengine.Project{Name: configs.GetProjectName(), Conf: configs.GetCurrentProjectConfig()}
	dir := paths.GetExecDirPath() + "/projects/" + project.Name + "/backup/snapshot"
	name := restore.MatchName(create.OrderedNames(dir), args.Name)
	if name == "" {
		fail(fmt.Sprintf("No snapshot named %q", args.Name))
	}
	if err := create.Remove(project, dir, name); err != nil {
		fail(err.Error())
	}
	if w != nil {
		w.Print(RemovedJSON{Name: name})
		return
	}
	fmtc.SuccessLn("Removed snapshot " + name)
}
