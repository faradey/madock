package docker

import (
	"fmt"
	"os"
	"path/filepath"
)

// Where a project's composer home comes from.
//
// madock has always pointed every project's `aruntime/projects/<name>/composer`
// at one directory — `~/.composer` — and it does so on every start: if it finds
// a real directory at that path it replaces it with the symlink. One cache and
// one set of globally installed packages for the whole machine.
//
// That is a deliberate trade and mostly a good one — the download cache is the
// bulk of it, 83 MB on the machine this was written on. What it also shares is
// the **global install**: `~/.composer/composer.json` and `vendor/`. Anything
// installed with `composer global require` in one project's container is the
// version every other project runs, whatever their own configuration says.
//
// Measured on 2026-09-01, and the reason these keys exist: madock-pro pins the
// Deployer version per project (`deploy/deployer_version`, written into each
// project's compose file). Moving ONE project from `^7` to `^8` moved all seven
// on that host — `dep --version` inside a container whose own compose line said
// `deployer/deployer:^7` answered `Deployer 8.0.5`. The pin is not wrong in the
// file; it simply cannot hold, because the install it pins is shared. One of
// those projects carried a recipe that only parses under v7, so its next deploy
// would have failed with nothing in its own configuration to explain why.
//
// So the sharing is now two decisions instead of one:
//
//	php/composer/shared_home   the global install — vendor/ and composer.json
//	php/composer/shared_cache  the download cache
//
// Both default to true, which is exactly the behaviour described above: nothing
// changes for anybody who does not ask. `shared_home=false` with
// `shared_cache=true` is the combination worth having — each project gets its
// own tools at its own versions, and downloads are still fetched once.

// prepareProjectComposerDir puts the project's composer directory into the
// shape the two settings ask for, and reports what it did in a line the caller
// can print (empty when nothing needed doing).
//
// It never touches the global directory itself: what changes is only the entry
// at projectDir — a symlink to the global home, or a real directory of the
// project's own.
func prepareProjectComposerDir(projectDir, globalDir string, sharedHome, sharedCache bool) (string, error) {
	if sharedHome {
		return linkToGlobal(projectDir, globalDir)
	}
	return isolateFromGlobal(projectDir, globalDir, sharedCache)
}

// linkToGlobal is the long-standing behaviour: the project directory IS the
// global one, reached through a symlink.
func linkToGlobal(projectDir, globalDir string) (string, error) {
	fi, err := os.Lstat(projectDir)
	if err != nil {
		return "", os.Symlink(globalDir, projectDir)
	}
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		return "", nil
	}

	// A real directory here is drift — normally it is the empty one madock
	// itself has just created. When it is not empty, somebody or something put
	// packages there, and they are about to go. Deleting them is still the
	// right move (the invariant is that this path is a link), but doing it in
	// silence is how "it reinstalled itself again" becomes a mystery.
	notice := ""
	if entries, readErr := os.ReadDir(projectDir); readErr == nil && len(entries) > 0 {
		notice = fmt.Sprintf(
			"composer: %s held %d entr(y/ies) of its own and has been replaced by the shared home at %s.\n"+
				"  Set php/composer/shared_home=false to keep a composer home per project.",
			projectDir, len(entries), globalDir)
	}
	if err := os.RemoveAll(projectDir); err != nil {
		return "", err
	}
	return notice, os.Symlink(globalDir, projectDir)
}

// isolateFromGlobal gives the project a composer home of its own, optionally
// keeping the download cache shared.
//
// Removing the symlink removes the link, never its target — os.Remove on a
// symlink does not follow it. The global home, and every other project pointing
// at it, are untouched.
func isolateFromGlobal(projectDir, globalDir string, sharedCache bool) (string, error) {
	notice := ""
	fi, err := os.Lstat(projectDir)
	switch {
	case err != nil:
		if mkErr := os.MkdirAll(projectDir, 0o777); mkErr != nil {
			return "", mkErr
		}
	case fi.Mode()&os.ModeSymlink == os.ModeSymlink:
		if rmErr := os.Remove(projectDir); rmErr != nil {
			return "", rmErr
		}
		if mkErr := os.MkdirAll(projectDir, 0o777); mkErr != nil {
			return "", mkErr
		}
		// Worth saying out loud: the project starts with an empty composer home,
		// so whatever was installed globally — `dep` above all — is installed
		// again on the next container start, at this project's own version.
		notice = fmt.Sprintf(
			"composer: %s no longer points at the shared home. Globally installed tools\n"+
				"  will be reinstalled here, at the versions this project asks for.", projectDir)
	}

	cache := filepath.Join(projectDir, "cache")
	if !sharedCache {
		return notice, nil
	}
	// The half worth keeping shared. A link one level down, so the project owns
	// composer.json and vendor/ while downloads are still fetched once per
	// machine.
	if cfi, cerr := os.Lstat(cache); cerr == nil {
		if cfi.Mode()&os.ModeSymlink == os.ModeSymlink {
			return notice, nil
		}
		// A real cache directory of its own: leave it. It is a cache, it costs
		// disk and nothing else, and deleting somebody's files to save space is
		// not this function's decision to make.
		return notice, nil
	}
	if err := os.MkdirAll(projectDir, 0o777); err != nil {
		return notice, err
	}
	return notice, os.Symlink(filepath.Join(globalDir, "cache"), cache)
}
