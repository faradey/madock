package embedded

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/faradey/madock/v4/src/helper/paths"
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

	written := map[string]bool{}
	if DockerFS != nil {
		extractFS(DockerFS, filepath.Join(execDir, "docker"), written)
	}
	if ScriptsFS != nil {
		extractFS(ScriptsFS, filepath.Join(execDir, "scripts"), written)
	}

	removeWithdrawn(execDir, written)
	writeManifest(execDir, written)

	os.WriteFile(markerFile, []byte(appVersion), 0644)
}

// manifestFile lists what the last extraction put on disk.
const manifestFile = ".embedded_files"

// removeWithdrawn deletes files a previous extraction wrote and this one did
// not — the ones a release withdrew.
//
// Extraction only ever added and overwrote, so a template dropped from the
// shipped set stayed on the machine for good. Measured on extmag: twelve
// `docker-compose.{darwin,linux,windows}.yml` files that neither madock 3.9.14
// nor madock-pro ships, and `snippets/dockerfile/php/nodejs` still carrying the
// `.php.nodejs.enabled` syntax that 3.9.8 removed. Harmless there only by
// accident — the first is empty and nothing includes the second any more.
//
// What makes it worth fixing before it stops being harmless: the resolver
// reaches `{execDir}/docker/{platform}/docker-compose.<GOOS>.yml` and applies
// what it finds as `docker-compose.override.yml`. A non-empty template withdrawn
// in a release would therefore go on applying on upgraded machines and not on
// fresh ones — at the same version. The result would depend on the history of
// the installation rather than on what it says it is.
//
// **Only files this mechanism itself wrote are removed.** The list comes from
// the manifest of the previous run, never from a walk of the directory:
// madock-pro extracts its own platform templates into the same tree, and a
// sweep of everything-not-in-the-embed would delete them. An installation with
// no manifest yet — every installation before this version — has nothing
// removed, so orphans predating it need one clean by hand.
func removeWithdrawn(execDir string, written map[string]bool) {
	body, err := os.ReadFile(filepath.Join(execDir, manifestFile))
	if err != nil {
		return
	}

	for _, previous := range strings.Split(string(body), "\n") {
		previous = strings.TrimSpace(previous)
		if previous == "" || written[previous] {
			continue
		}
		// Relative paths only, and nothing that climbs out of the tree: the
		// manifest is a file on disk and this deletes what it names.
		if filepath.IsAbs(previous) || strings.Contains(previous, "..") {
			continue
		}
		os.Remove(filepath.Join(execDir, previous))
	}
}

// writeManifest records what this extraction wrote, so the next one can tell a
// withdrawn file from somebody else's.
func writeManifest(execDir string, written map[string]bool) {
	paths := make([]string, 0, len(written))
	for path := range written {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	os.WriteFile(filepath.Join(execDir, manifestFile), []byte(strings.Join(paths, "\n")+"\n"), 0644)
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

func extractFS(fsys fs.FS, destDir string, written map[string]bool) {
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
		if rel, relErr := filepath.Rel(paths.GetExecDirPath(), target); relErr == nil {
			written[filepath.ToSlash(rel)] = true
		}
		return os.WriteFile(target, data, 0755)
	})
}
