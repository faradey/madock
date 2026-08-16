package embedded

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/faradey/madock/v3/src/helper/paths"
)

var DockerFS fs.FS
var ScriptsFS fs.FS

func SetDockerFS(f fs.FS) {
	DockerFS = f
}

func SetScriptsFS(f fs.FS) {
	ScriptsFS = f
}

// ExtractIfNeeded extracts embedded assets to disk when version changes.
//
// Never over a source checkout. The extraction exists for an installation made
// from a downloaded binary, which has no other way to get its templates; an
// installation made by install.sh is a clone, where docker/ is tracked and
// arrives with git pull. There the two are the same directory, so extracting
// writes a build-time snapshot over the working copy — and a binary built
// before an edit silently reverts it.
//
// That is not hypothetical: it undid three edited templates mid-session, and a
// test then passed against the reverted files while reporting the number the
// engine had picked for itself. `version` is enough to trigger it, because the
// check runs before any command.
func ExtractIfNeeded(appVersion string) {
	execDir := paths.GetExecDirPath()

	if isSourceCheckout(execDir) {
		return
	}

	markerFile := filepath.Join(execDir, ".embedded_version")

	existing, _ := os.ReadFile(markerFile)
	if string(existing) == appVersion {
		return
	}

	if DockerFS != nil {
		extractFS(DockerFS, filepath.Join(execDir, "docker"))
	}
	if ScriptsFS != nil {
		extractFS(ScriptsFS, filepath.Join(execDir, "scripts"))
	}

	os.WriteFile(markerFile, []byte(appVersion), 0644)
}

// isSourceCheckout reports whether the installation directory is madock's own
// source tree.
//
// go.mod is the test because it is what tells the two installations apart:
// install.sh clones the repository, so the templates there are files git owns,
// while a release binary is unpacked on its own with nothing beside it.
func isSourceCheckout(execDir string) bool {
	_, err := os.Stat(filepath.Join(execDir, "go.mod"))
	return err == nil
}

func extractFS(fsys fs.FS, destDir string) {
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "." {
			return err
		}
		target := filepath.Join(destDir, path)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		os.MkdirAll(filepath.Dir(target), 0755)
		return os.WriteFile(target, data, 0755)
	})
}
