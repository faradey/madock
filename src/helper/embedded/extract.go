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

// sourceSentinel is the file that says the docker tree in this directory is
// source rather than an extraction: the embed declaration itself. It is listed
// in no //go:embed pattern — those name asset directories — so it exists in a
// clone and never in an extracted tree.
const sourceSentinel = "docker/embed.go"

// isSourceCheckout reports whether the docker tree in the installation
// directory is source that git delivers, rather than one extraction is
// responsible for keeping current.
//
// The question is about the templates, so the test is a file in the template
// tree — not go.mod, which was the first answer and the wrong one. go.mod says
// "this directory is a Go module", and that is true of madock-pro as well:
// pro's installation is a clone with go.mod at the root, but its docker/ is in
// .gitignore on purpose, because the assets belong to the imported madock
// module and arrive by extraction. Testing go.mod therefore switched extraction
// off in the one installation where nothing else brings the templates in, and
// that tree simply stopped moving — measured 2026-08-17, `.embedded_version`
// read 3.6.7 against a 3.9.3 module, with 47 templates still in the syntax the
// engine replaced in 3.9.1. Nothing announced it: the renderer converts the old
// syntax on the fly, so everything worked, from templates two years old.
//
// `docker/embed.go` is the honest signal. It is the embed declaration, so it
// exists wherever the tree is source, and it appears in no //go:embed pattern —
// those name asset directories — so extraction never writes it and cannot make
// an extracted tree look like a checkout.
func isSourceCheckout(execDir string) bool {
	_, err := os.Stat(filepath.Join(execDir, filepath.FromSlash(sourceSentinel)))
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
