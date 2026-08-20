package project

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/faradey/madock/v4/src/helper/paths"
)

// Fingerprint hashes everything MakeConf generates for a project: the compose
// files and the build context under them. It answers one question — were the
// running containers created from these files, or from an older render of them.
//
// Hashing the output rather than the config is deliberate. A config key that no
// template reads (an SSH host, a cron flag) changes the config but not the
// stack, and recreating containers for it would be noise. Only a difference
// docker would actually act on shows up here.
func Fingerprint(projectName string) string {
	pp := paths.NewProjectPaths(projectName)
	root := pp.RuntimeDir()

	// Nothing generated yet is not "an empty stack" — it is no answer at all.
	// Hashing the empty set here would give a perfectly stable value that could
	// then be recorded as the stack the containers were built from.
	if _, err := os.Stat(root); err != nil {
		return ""
	}

	type entry struct{ name, sum string }
	var entries []entry

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// Symlinks point at the project source, the global composer dir and
		// ~/.ssh. Following them would hash the entire working tree and make
		// the fingerprint change on every edit of the application.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		sum := sha256.Sum256(content)
		entries = append(entries, entry{name: filepath.ToSlash(rel), sum: hex.EncodeToString(sum[:])})
		return nil
	})
	if err != nil {
		return ""
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	h := sha256.New()
	for _, e := range entries {
		h.Write([]byte(e.name))
		h.Write([]byte{0})
		h.Write([]byte(e.sum))
		h.Write([]byte{0})
	}

	return hex.EncodeToString(h.Sum(nil))
}

// appliedFingerprintPath is where the fingerprint of the last successfully
// started stack is recorded.
func appliedFingerprintPath(projectName string) string {
	return filepath.Join(paths.MakeDirsByPath(paths.CacheDir()), strings.ToLower(projectName)+"-stack-hash")
}

// RecordApplied stores the current fingerprint as the one the running
// containers were created from. Called after a successful up.
func RecordApplied(projectName string) {
	fingerprint := Fingerprint(projectName)
	if fingerprint == "" {
		return
	}
	_ = os.WriteFile(appliedFingerprintPath(projectName), []byte(fingerprint), 0644)
}

// NeedsRecreate reports that the generated stack differs from the one the
// running containers were created from — `docker compose start` would bring up
// the old containers and quietly ignore the new files.
//
// An absent record means this project predates the fingerprint (or its cache
// was cleared). That is adopted silently rather than treated as a change:
// upgrading madock must not force a rebuild of every project on its next start.
//
// Both failure modes answer "no". An unreadable stack is not evidence that
// anything changed, and recreating containers on the strength of a stat error
// would turn a permissions problem into a rebuild of a working environment.
// The cost of being wrong this way is that a change goes unnoticed until the
// next rebuild — which is exactly where this started, so it is not a new trap.
func NeedsRecreate(projectName string) bool {
	fingerprint := Fingerprint(projectName)
	if fingerprint == "" {
		return false
	}

	stored, err := os.ReadFile(appliedFingerprintPath(projectName))
	if err != nil {
		RecordApplied(projectName)
		return false
	}

	return strings.TrimSpace(string(stored)) != fingerprint
}
