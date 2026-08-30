package paths

import (
	"os"
	"path/filepath"
)

// IsDirEmpty reports whether a directory holds anything worth refusing to
// install over.
//
// It guards `setup` against writing a platform into a directory somebody is
// already using, so what counts as "empty" is a judgement rather than a fact.
// Two things are deliberately not counted:
//
//   - `.madock`, because it is ours. A directory holding only madock's own
//     configuration is one where setup was started and not finished, and
//     refusing there would leave the person unable to continue without deleting
//     a file they did not create.
//   - directories that are themselves empty, all the way down. An empty tree
//     holds nothing to lose, and it is what a half-finished clone or an
//     interrupted extraction leaves behind.
//
// The second rule existed in exactly one of the six copies this replaces —
// medusa's — and nowhere else. Nothing announced the difference: five platforms
// simply refused to set up in a directory the sixth accepted.
//
// An unreadable directory answers "empty", which is the permissive direction and
// is what every copy already did. Refusing on a read error would block setup on
// a permissions problem the message would not explain.
func IsDirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return true
	}

	for _, entry := range entries {
		if entry.Name() == ".madock" || entry.Name() == "." || entry.Name() == ".." {
			continue
		}
		if entry.IsDir() && IsDirEmpty(filepath.Join(path, entry.Name())) {
			continue
		}

		return false
	}

	return true
}
