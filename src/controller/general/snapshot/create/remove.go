package create

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/paths"
	"github.com/faradey/madock/v4/src/helper/snapshotengine"
)

// OrderedNames lists the snapshot directories in dir, oldest first.
//
// By the timestamp at the end of the name, not by the name: `snapshot:create
// -n zzz` sorts after a newer `-n aaa` as a string.
func OrderedNames(dir string) []string {
	if !paths.IsFileExist(dir) {
		return nil
	}
	var names []string
	for _, n := range paths.GetDirs(dir) {
		names = append(names, filepath.Base(n))
	}
	sort.SliceStable(names, func(i, j int) bool {
		a, b := stamp(names[i]), stamp(names[j])
		if a != b {
			return a < b
		}
		return names[i] < names[j]
	})
	return names
}

// stamp is the "2006-01-02-15-04-05" suffix snapshot:create writes.
func stamp(name string) string {
	const width = len("2006-01-02-15-04-05")
	if len(name) < width {
		return name
	}
	return name[len(name)-width:]
}

// Remove deletes one snapshot: the engine's own snapshot first, then the
// directory.
//
// In that order and no other. A directory removed first leaves the engine's
// snapshot behind with nothing pointing at it — on zfs or btrfs it holds its
// space for good and no command lists it. And an engine that refuses to drop
// (zfs will, while a clone still lives on the snapshot) keeps the directory, so
// the snapshot stays restorable and visible instead of half gone.
//
// The project travels with its configuration: an engine finds its storage from
// it (snapshot/zfs/parent, snapshot/btrfs/path), and the first version passed
// the name alone — every engine snapshot would have been "kept" with a refusal
// about a missing setting.
func Remove(project snapshotengine.Project, dir, name string) error {
	path := filepath.Join(dir, name)
	marker, byEngine, err := snapshotengine.ReadMarker(path)
	if err != nil {
		return err
	}
	if byEngine {
		engine, ok := snapshotengine.ByName(marker.Engine)
		if !ok {
			return fmt.Errorf("%w: snapshot %s is held by %q, so it cannot be dropped by this build", snapshotengine.ErrUnknown, name, marker.Engine)
		}
		if err := engine.Drop(project, marker.Ref); err != nil {
			return fmt.Errorf("%s would not drop snapshot %s (a clone may still live on it): %w", marker.Engine, name, err)
		}
	}
	return os.RemoveAll(path)
}

// Keep reads snapshot/keep: how many snapshots a project holds. 0, empty or
// anything unreadable means no limit, which is how it has always been.
func Keep(projectConf map[string]string) int {
	n, err := strconv.Atoi(strings.TrimSpace(projectConf["snapshot/keep"]))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// Prune removes the oldest snapshots in dir until keep remain. One that cannot
// be removed — an engine snapshot a clone still lives on — is skipped and said,
// and the next oldest is tried, so a pinned snapshot does not stop the rest
// from being cleaned. It counts towards keep all the same: it is there.
func Prune(project snapshotengine.Project, dir string, keep int) (removed []string) {
	if keep <= 0 {
		return nil
	}
	names := OrderedNames(dir)
	excess := len(names) - keep
	for _, name := range names {
		if excess <= 0 {
			break
		}
		if err := Remove(project, dir, name); err != nil {
			fmtc.WarningLn("Kept snapshot " + name + ": " + err.Error())
			continue
		}
		removed = append(removed, name)
		excess--
	}
	return removed
}
