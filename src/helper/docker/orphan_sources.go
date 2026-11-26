package docker

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/faradey/madock/v4/src/helper/paths"
)

// Orphans that are not docker's.
//
// A removal that stops halfway leaves things docker's labels know nothing
// about: project:remove ends with RemoveAll on projects/<name>/, and a file it
// cannot delete keeps the directory while config.xml — the registry entry —
// is already gone. After that no command named the directory. madock-pro keeps
// project data outside docker too (a zfs dataset per project), and only it
// knows where; it adds those through RegisterOrphanSource.
//
// Listed only. `project:orphans --remove` still takes docker resources and
// nothing else: a directory or a dataset is somebody's data until a person has
// looked at it.

// OrphanSource lists orphans madock itself cannot see. It is given the names
// the registry holds. An error means the source could not tell — not that it
// found nothing — and project:orphans says so instead of answering "nothing".
type OrphanSource func(known map[string]bool) ([]Orphan, error)

type namedSource struct {
	name string
	find OrphanSource
}

var orphanSources []namedSource

// RegisterOrphanSource adds a source to project:orphans. name says what it
// looks at ("zfs datasets") and appears in the answer when it fails.
//
// Extension point for madock-pro. Call from init().
func RegisterOrphanSource(name string, find OrphanSource) {
	orphanSources = append(orphanSources, namedSource{name: name, find: find})
}

// OrphanUnknown is a source that could not answer.
type OrphanUnknown struct {
	Source string `json:"source"`
	Error  string `json:"error"`
}

// FindAllOrphans is FindOrphans plus the project directories with no registry
// entry and every registered source. unknown names each source that could not
// tell; it is never folded into "nothing found".
func FindAllOrphans(known map[string]bool) (found []Orphan, unknown []OrphanUnknown) {
	found = FindOrphans(known)

	dirs, err := orphanProjectDirs(paths.GetExecDirPath(), known)
	if err != nil {
		unknown = append(unknown, OrphanUnknown{Source: "project directories", Error: err.Error()})
	}
	found = append(found, dirs...)

	more, notTold := fromSources(orphanSources, known)
	found = append(found, more...)
	unknown = append(unknown, notTold...)

	sortOrphans(found)
	return found, unknown
}

// fromSources asks every registered source. A source that fails is named in
// unknown, and what it did list is kept.
func fromSources(sources []namedSource, known map[string]bool) (found []Orphan, unknown []OrphanUnknown) {
	for _, source := range sources {
		more, err := source.find(known)
		if err != nil {
			unknown = append(unknown, OrphanUnknown{Source: source.name, Error: err.Error()})
		}
		found = append(found, more...)
	}
	return found, unknown
}

// supportDirs are what projects/ holds beside the projects, as ListProjectsIn
// says: not entries and not litter.
var supportDirs = map[string]bool{"bin": true, "docker": true, "assets": true}

// orphanProjectDirs lists projects/<name>/ directories with no config.xml —
// the registry entry is that file — whose name the registry does not hold.
//
// A name the registry holds is never listed, whatever its directory looks like:
// an entry that resolves to nothing is project:list --stale's, not this. A
// projects/ that does not exist is an installation with no projects, not an
// error; one that cannot be read is.
func orphanProjectDirs(execDir string, known map[string]bool) ([]Orphan, error) {
	root := filepath.Join(execDir, "projects")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []Orphan
	for _, entry := range entries {
		name := entry.Name()
		if known[name] || supportDirs[name] {
			continue
		}
		dir := filepath.Join(root, name)
		// Stat, not entry.IsDir(): a symlink to a directory is a directory here,
		// as it is to ListProjectsIn. One that resolves to nothing is the
		// registry's broken-link entry, and is not this.
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		if paths.IsFileExist(filepath.Join(dir, "config.xml")) {
			continue
		}
		out = append(out, Orphan{Kind: "directory", Name: dir, Project: name})
	}
	return out, nil
}

func sortOrphans(out []Orphan) {
	sort.Slice(out, func(i, j int) bool {
		if out[i].Project != out[j].Project {
			return out[i].Project < out[j].Project
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Name < out[j].Name
	})
}
