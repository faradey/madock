package embedded

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/faradey/madock/v4/src/helper/paths"
)

var DockerFS fs.FS
var ScriptsFS fs.FS

// SetDockerFS installs the template tree an installation unpacks.
//
// Extension point for madock-pro, and the most load-bearing one there is: pro's
// own `main` sets both of these before anything runs, which is how a pro
// installation ends up with pro's templates rather than community's. Nothing in
// this module calls either — the binary that does is the other one.
func SetDockerFS(f fs.FS) {
	DockerFS = f
}

// SetScriptsFS installs the script tree an installation unpacks.
//
// Extension point for madock-pro: same seam as SetDockerFS, same caller, same
// reason it is unreachable from here. Spelled out rather than cross-referenced,
// because the phrase is what a search finds and a pointer to a neighbour is not.
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
	var failures []ExtractFailure
	if DockerFS != nil {
		failures = append(failures, extractFS(DockerFS, filepath.Join(execDir, "docker"), written)...)
	}
	if ScriptsFS != nil {
		failures = append(failures, extractFS(ScriptsFS, filepath.Join(execDir, "scripts"), written)...)
	}

	// An incomplete extraction is not allowed to draw any conclusion from what
	// it managed to write, and all three of the steps below would.
	//
	// `removeWithdrawn` is the dangerous one: it deletes what the previous
	// manifest names and this run did not write, so a truncated run reads its
	// own gap as "the release withdrew these" and removes templates that are
	// still shipped and still on disk. A partial extraction would stop being a
	// tree missing what it never wrote and become one actively stripped of what
	// worked yesterday.
	//
	// `writeManifest` would then publish that gap as the record of what
	// extraction owns, so the next run — even a healthy one — inherits a
	// manifest that has forgotten two thirds of the tree.
	//
	// And the version stamp would make it stick: `ExtractIfNeeded` returns early
	// when the marker equals the running version, so stamping a run that failed
	// declares the installation current and nothing retries it. Leaving it
	// unstamped is what makes the tree repair itself the moment the permissions
	// are fixed, at the cost of the warning repeating until then — which is the
	// right way round for a fault nothing else reports.
	if len(failures) > 0 {
		reportFailures(os.Stderr, failures, execDir)
		return
	}

	removeWithdrawn(execDir, written)
	writeManifest(execDir, written)

	os.WriteFile(markerFile, []byte(appVersion), 0644)
}

// ExtractFailure is one file the extraction could not put on disk.
type ExtractFailure struct {
	// Path is relative to the installation directory, slash-separated —
	// `docker/snippets/…`, the way the rest of madock names a template.
	Path string
	Err  error
}

// reportFailures says what did not reach the disk, and why.
//
// On stderr, for the reason the drift warning is: `--json` output has to stay
// parseable, and this one repeats on every command until somebody fixes the
// permissions.
func reportFailures(w io.Writer, failures []ExtractFailure, execDir string) {
	if len(failures) == 0 {
		return
	}

	sort.Slice(failures, func(i, j int) bool { return failures[i].Path < failures[j].Path })

	fmt.Fprintf(w, "⚠ %d of madock's own templates could not be written into %s — the tree is incomplete\n",
		len(failures), execDir)

	const named = 3
	for i, failure := range failures {
		if i == named {
			break
		}
		fmt.Fprintf(w, "  %s: %v\n", failure.Path, reason(failure.Err))
	}
	if remainder := len(failures) - named; remainder > 0 {
		fmt.Fprintf(w, "  and %d more\n", remainder)
	}

	fmt.Fprintln(w, "Nothing was deleted and the version stamp was left alone, so the next madock command tries again.")
	fmt.Fprintln(w, "Until then a build can stop at \"no such file or directory\" for a template that should exist.")
}

// reason strips the wrapper that already names the path, so the line does not
// carry it twice: an *fs.PathError prints as `open /opt/…/worker.yml:
// permission denied`, and the path is the first thing on the line already.
func reason(err error) error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err
	}

	return err
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

// extractFS writes one embedded tree to disk and returns what it could not
// write, having written everything else.
//
// **The callback never returns an error, and that is the whole point.**
// `fs.WalkDir` reads any non-nil answer as "stop the walk", and this one used to
// return `os.WriteFile`'s directly while `extractFS` discarded the walk's
// result — so a single file that could not be written ended the extraction
// where it stood, silently, and everything lexically after it never reached the
// disk.
//
// Measured on the `shopify-e2e` machine on 2026-08-27: exactly one file in the
// tree, `docker/snippets/docker-compose/worker.yml`, was left owned by root by a
// single run under `MADOCK_USER=root` five days earlier. Every extraction since
// stopped on it — 202 files of 309 — and nothing about the installation looked
// wrong: the directories exist, because the previous pass created them before
// the failing write; `.embedded_version` was stamped, because it is written
// after this returns and this returned normally; `status` and `project:list`
// work, because they need no templates. The only symptom was a build printing
// three paths it had looked for a snippet in, and it cost an hour to reach the
// cause.
//
// A partial extraction that says nothing is worse than one that stops: the one
// that stops gets fixed the same minute.
func extractFS(fsys fs.FS, destDir string, written map[string]bool) []ExtractFailure {
	var failures []ExtractFailure

	// The path a person can act on is the one in the installation, not the one
	// inside the embed — they differ by the tree's name (`docker`, `scripts`).
	fail := func(path string, err error) {
		named := path
		if rel, relErr := filepath.Rel(paths.GetExecDirPath(), filepath.Join(destDir, path)); relErr == nil {
			named = filepath.ToSlash(rel)
		}
		failures = append(failures, ExtractFailure{Path: named, Err: err})
	}

	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fail(path, err)
			return nil
		}
		if path == "." {
			return nil
		}
		target := filepath.Join(destDir, path)
		if d.IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				fail(path, err)
				// fs.SkipDir is the one non-nil answer WalkDir does not treat as
				// a failure, so the rest of the tree still gets written. Without
				// it every file under an unreachable directory is reported
				// separately, and one bad directory becomes a screen of noise
				// saying a single thing.
				return fs.SkipDir
			}
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			fail(path, err)
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			fail(path, err)
			return nil
		}
		if err := writeExtracted(target, data); err != nil {
			fail(path, err)
			return nil
		}
		// Recorded after the write, never before. The manifest is what
		// `removeWithdrawn` deletes by, so a file claimed and not written is a
		// file the next run believes extraction owns.
		if rel, relErr := filepath.Rel(paths.GetExecDirPath(), target); relErr == nil {
			written[filepath.ToSlash(rel)] = true
		}
		return nil
	})

	return failures
}

// writeExtracted writes one template, replacing the file outright when it cannot
// be written in place.
//
// The case this exists for is the one that was found on disk: a run under
// `MADOCK_USER=root` leaves root-owned files in the installation, and every
// later extraction as the ordinary user is refused on them for good. Unlink
// permission on Unix belongs to the containing directory rather than to the
// file, so removing it and writing afresh succeeds wherever the directory is
// ours — the tree repairs itself instead of needing a chown nobody knows to run.
//
// Only on a permission error, and deliberately so. A write that failed for any
// other reason has not told us that replacing the file would help, and removing
// it first would turn "could not update this template" into "this template is
// gone".
func writeExtracted(target string, data []byte) error {
	err := os.WriteFile(target, data, 0755)
	if err == nil || !errors.Is(err, fs.ErrPermission) {
		return err
	}

	if removeErr := os.Remove(target); removeErr != nil {
		// The write error is the one worth reporting: it says what was refused.
		return err
	}

	return os.WriteFile(target, data, 0755)
}
