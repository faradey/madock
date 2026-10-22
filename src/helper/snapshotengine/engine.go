// Package snapshotengine is where a project's database volumes are snapshotted,
// restored and cloned by something other than tar.
//
// madock itself copies them with tar through a helper container: the project
// stands still for the length of the copy, and the copy is as large and as slow
// as the data. A filesystem with copy-on-write snapshots does the same in
// milliseconds, atomically, with the database running — measured on 2026-10-17,
// a 2.2 GB MariaDB took 128 s through tar and 56 ms as a ZFS snapshot, and came
// back intact from snapshots taken under a write load. That needs root, a pool
// and a volume layout madock does not set up, so the implementations live
// elsewhere (madock Pro) and register here; madock only asks.
//
// The choice is the project's `snapshot/engine` key. Empty or `tar` is madock's
// own path, unchanged. A name nothing registered is refused: a project
// configured for zfs and run by a binary that has no zfs must not quietly fall
// back to a copy that stops the project and takes minutes.
//
// The database volumes go through an engine — dbdata and, when enabled,
// dbdata2. The project's other named volumes (Volumes) go through it too when it
// says so (AllVolumes); otherwise madock copies them with tar. Project files keep
// going through tar.
package snapshotengine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Project is the project an engine is asked about.
type Project struct {
	Name string
	Conf map[string]string
}

// Engine snapshots, restores and clones a project's database volumes, and all of
// its named volumes when it also implements AllVolumes.
type Engine interface {
	// Live reports whether Capture can run with the project's database up.
	// Only an atomic snapshot can: one instant for every file. Anything that
	// copies file by file must not, because InnoDB writes while it copies.
	Live() bool

	// Capture takes snapshot `name` of the project's database volumes and
	// returns a reference Restore will be handed back.
	Capture(p Project, name string) (ref string, err error)

	// Restore puts the database volumes back as they were at ref. The project's
	// containers are down when it is called.
	Restore(p Project, ref string) error

	// Clone gives dst database volumes holding what src's hold now. dst's
	// containers do not exist yet; Clone runs before they are created.
	Clone(src, dst Project) error

	// Drop removes the snapshot behind ref. A snapshot directory removed by
	// hand leaves the engine's snapshot behind; this is how it goes too.
	Drop(p Project, ref string) error
}

var (
	mu      sync.RWMutex
	engines = map[string]Engine{}
)

// Register makes an engine selectable by name. A second registration under the
// same name is a programming error.
func Register(name string, e Engine) {
	mu.Lock()
	defer mu.Unlock()
	if _, dup := engines[name]; dup {
		panic("snapshotengine: " + name + " registered twice")
	}
	engines[name] = e
}

// Names lists the registered engines, sorted.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(engines))
	for n := range engines {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// ErrUnknown is a configured engine that nothing registered.
var ErrUnknown = errors.New("snapshot engine is not available in this build")

// For returns the project's engine. nil with a nil error means madock's own
// tar path. A configured name nothing registered is ErrUnknown, wrapped with
// the name and what is available.
func For(conf map[string]string) (Engine, string, error) {
	name := strings.TrimSpace(conf["snapshot/engine"])
	if name == "" || name == "tar" {
		return nil, "tar", nil
	}
	mu.RLock()
	e, ok := engines[name]
	mu.RUnlock()
	if !ok {
		available := append([]string{"tar"}, Names()...)
		return nil, name, fmt.Errorf("%w: snapshot/engine is %q; available: %s", ErrUnknown, name, strings.Join(available, ", "))
	}
	return e, name, nil
}

// AllVolumes is an engine that captures, restores and clones every volume in
// Volumes, not only the databases. One that does not leaves the others to
// madock's tar, volume by volume.
type AllVolumes interface {
	AllVolumes() bool
}

// SnapshotCloner is an engine that can give a new project the volumes of one of
// its snapshots, not only of the source as it is now — `project:clone --from`.
// On zfs and btrfs that is a clone of the snapshot itself: nothing is copied
// and the source is not asked anything. An engine without it refuses --from
// for the snapshots it holds.
type SnapshotCloner interface {
	CloneSnapshot(src Project, ref string, dst Project) error
}

// CodeHolder is an engine that can keep the project's code in its storage too.
// CodeDir answers where, for this project; "" means the code is not there and
// madock archives the files with tar as before.
//
// With the code in the same dataset as the volumes, one atomic snapshot holds
// the code, the databases and every volume at one instant, and a clone gets its
// code from the clone of the dataset: nothing is copied. Measured 2026-10-18 on
// btrfs: ten clones of a Magento checkout in 84 ms and 50 MB, against 26 s and
// 3.8 GB with cp.
type CodeHolder interface {
	CodeDir(p Project) string
}

// CodeDirOf is e's CodeDir for p, or "" when e keeps no code.
func CodeDirOf(e Engine, p Project) string {
	if h, ok := e.(CodeHolder); ok {
		return h.CodeDir(p)
	}
	return ""
}

// TakesAll reports whether e handles every volume itself.
func TakesAll(e Engine) bool {
	a, ok := e.(AllVolumes)
	return ok && a.AllVolumes()
}

// Stops reports whether a snapshot with e must stop the project's containers.
// snapshot/live=stop asks for it even of an engine that could run live.
//
// So does a volume the engine leaves to tar. A search index copied file by file
// out of a running OpenSearch is torn the way a database would be; an engine
// that snapshots the databases atomically does not make the tar copy beside it
// safe, so a project with such volumes stands still unless its engine takes
// them too.
func Stops(e Engine, conf map[string]string) bool {
	if e == nil {
		return true
	}
	if strings.TrimSpace(conf["snapshot/live"]) == "stop" {
		return true
	}
	if !TakesAll(e) && len(Others(conf)) > 0 {
		return true
	}
	return !e.Live()
}

// Marker is what a snapshot directory holds instead of db.tar.gz when an engine
// took the databases. snapshot:list shows it as a file like any other, and
// snapshot:restore chooses its path by whether it is there — so snapshots taken
// before engines existed restore exactly as they always did.
type Marker struct {
	Engine string `json:"engine"`
	Ref    string `json:"ref"`
}

// MarkerFile is the marker's name inside a snapshot directory.
const MarkerFile = "engine.json"

// WriteMarker records which engine holds the databases of the snapshot in dir.
func WriteMarker(dir string, m Marker) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, MarkerFile), append(data, '\n'), 0o644)
}

// ReadMarker reads the marker in dir. ok is false when there is none, which is
// every snapshot taken with tar.
func ReadMarker(dir string) (Marker, bool, error) {
	data, err := os.ReadFile(filepath.Join(dir, MarkerFile))
	if errors.Is(err, os.ErrNotExist) {
		return Marker{}, false, nil
	}
	if err != nil {
		return Marker{}, false, err
	}
	var m Marker
	if err := json.Unmarshal(data, &m); err != nil {
		return Marker{}, false, fmt.Errorf("%s: %w", filepath.Join(dir, MarkerFile), err)
	}
	if m.Engine == "" || m.Ref == "" {
		return Marker{}, false, fmt.Errorf("%s names no engine or no snapshot", filepath.Join(dir, MarkerFile))
	}
	return m, true, nil
}

// ByName returns a registered engine by name, for a snapshot whose marker names
// it — which may differ from what the project is configured for now.
func ByName(name string) (Engine, bool) {
	mu.RLock()
	defer mu.RUnlock()
	e, ok := engines[name]
	return e, ok
}

// reset empties the registry; tests only.
func reset() {
	mu.Lock()
	engines = map[string]Engine{}
	mu.Unlock()
}

// Sizer is an engine that can say how much one of its snapshots holds.
//
// Extension point for madock-pro: a snapshot an engine took leaves only
// engine.json in its directory, so the size madock sums from the directory is
// the size of that marker — a few dozen bytes for a snapshot of gigabytes
// (Indemorio IND-15). An engine that implements this reports the real figure,
// which --json gives as engine_bytes.
type Sizer interface {
	SnapshotBytes(p Project, ref string) (int64, error)
}

// BytesOf is e's SnapshotBytes for ref, and whether it could say.
func BytesOf(e Engine, p Project, ref string) (int64, bool) {
	s, ok := e.(Sizer)
	if !ok {
		return 0, false
	}
	n, err := s.SnapshotBytes(p, ref)
	if err != nil {
		return 0, false
	}
	return n, true
}
