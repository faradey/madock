package clone

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// cloneTarget is where a clone's code goes, and what a failed clone may remove.
type cloneTarget struct {
	// Dir is the clone's project directory: <parent>/<name>.
	Dir string
	// Created is the topmost directory this run has to create to reach Dir —
	// Dir itself, or a parent that did not exist yet. "" when Dir is already
	// there: an existing directory is somebody's, and a failed clone leaves it.
	Created string
}

// resolveTarget decides the clone's directory.
//
// Without --path the clone lands beside the source, as it always has. Test
// runs clone a project dozens of times, and beside the source meant beside a
// repository checkout: senge's throwaway clones ended up next to its sources.
// --path names the directory the clone goes into instead. It is a parent, so
// the clone's directory is still named after the clone — the name the project,
// its containers and its hosts carry. A relative --path is taken against the
// directory madock was run from, a leading ~/ against the home directory, and
// directories that do not exist yet are created.
//
// Refused, before anything is made:
//   - a target that exists and is not an empty directory: the clone unpacks its
//     code with `rm -rf /var/www/html/*` over the mount, which would delete
//     whatever was there;
//   - a target inside the source's own directory: the source's code mount would
//     then hold the clone, and the next clone or snapshot of the source would
//     carry it along.
func resolveTarget(sourcePath, parent, name, cwd, home string) (cloneTarget, error) {
	parent = strings.TrimSpace(parent)
	if parent == "" {
		parts := strings.Split(sourcePath, "/")
		dir := strings.Join(parts[:len(parts)-1], "/") + "/" + name
		return cloneTarget{Dir: dir, Created: missingRoot(dir)}, nil
	}

	switch {
	case parent == "~":
		parent = home
	case strings.HasPrefix(parent, "~/"):
		parent = filepath.Join(home, parent[2:])
	}
	if strings.HasPrefix(parent, "~") {
		return cloneTarget{}, fmt.Errorf("--path %q: only ~/ (this user's home) is understood, not another user's", parent)
	}
	if !filepath.IsAbs(parent) {
		parent = filepath.Join(cwd, parent)
	}
	dir := filepath.Join(filepath.Clean(parent), name)

	if src := filepath.Clean(sourcePath); src != "" && src != "/" {
		if dir == src || strings.HasPrefix(dir, src+string(filepath.Separator)) {
			return cloneTarget{}, fmt.Errorf("--path puts the clone at %s, inside the source project %s; the source's code mount would then carry the clone", dir, src)
		}
	}

	info, err := os.Lstat(dir)
	switch {
	case os.IsNotExist(err):
	case err != nil:
		return cloneTarget{}, err
	case !info.IsDir():
		return cloneTarget{}, fmt.Errorf("%s already exists and is not a directory", dir)
	default:
		entries, err := os.ReadDir(dir)
		if err != nil {
			return cloneTarget{}, err
		}
		if len(entries) > 0 {
			return cloneTarget{}, fmt.Errorf("%s already exists and is not empty; the clone would replace what is in it", dir)
		}
	}

	return cloneTarget{Dir: dir, Created: missingRoot(dir)}, nil
}

// missingRoot is the topmost directory on the way to dir that does not exist,
// or "" when dir exists.
func missingRoot(dir string) string {
	if _, err := os.Lstat(dir); err == nil {
		return ""
	}
	top := dir
	for {
		up := filepath.Dir(top)
		if up == top {
			return top
		}
		if _, err := os.Lstat(up); err == nil {
			return top
		}
		top = up
	}
}
