package embedded

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/faradey/madock/v4/src/helper/paths"
)

// Drift is the disagreement between the templates a binary was built from and
// the ones it is about to render.
type Drift struct {
	// Changed exists in both and differs.
	Changed []string
	// Missing is in the binary and not on disk.
	Missing []string
	// Extra is on disk and not in the binary.
	Extra []string
}

// Total counts everything that disagrees.
func (d *Drift) Total() int {
	if d == nil {
		return 0
	}

	return len(d.Changed) + len(d.Missing) + len(d.Extra)
}

// Examples names a few of the differing paths, for a message that has to fit on
// a screen. Missing and Extra come first: a changed template usually still
// renders, while one that has moved is what stops a build.
func (d *Drift) Examples(limit int) []string {
	if d == nil {
		return nil
	}

	out := make([]string, 0, limit)
	for _, group := range [][]string{d.Missing, d.Extra, d.Changed} {
		for _, path := range group {
			if len(out) == limit {
				return out
			}
			out = append(out, path)
		}
	}

	return out
}

// reported keeps the warning to one appearance per run: a command may render
// several times (project templates, then nginx's) and the answer does not
// change between them.
var reported sync.Once

// ReportOnce writes the warning to stderr, at most once per run.
//
// **Called from where templates are rendered, and from rebuild's pre-flight —
// not from the dispatcher.** It used to run before every command, which was
// accurate and unusable: on a machine where somebody is editing templates, every
// `db:execute`, `cli` and `setup:upgrade` in every other project carried a
// warning about a binary its user was not going to rebuild. A warning that is
// always there is one that stops being read, and this one exists for a failure
// that took out every environment on the machine once already.
//
// The hazard is rendering, so the warning belongs where rendering happens. The
// exception is rebuild, which destroys the containers first and generates the
// build context afterwards — there it has to be said in the pre-flight, before
// anything is taken down, or it arrives after the damage.
//
// stderr, not stdout: `--json` output has to stay parseable.
func ReportOnce() {
	reported.Do(func() {
		Report(os.Stderr, TemplateDrift(), paths.GetExecDirPath())
	})
}

// Report writes the warning for a drift, and nothing at all when there is none.
//
// Takes a writer so a test can say which stream it went to, and a drift so it
// can be given one that did not have to be produced by a real installation.
func Report(w io.Writer, drift *Drift, execDir string) {
	if drift == nil {
		return
	}

	fmt.Fprintf(w, "⚠ This binary was built from different templates than the ones in this tree — "+
		"%d changed, %d missing from disk, %d new to it\n",
		len(drift.Changed), len(drift.Missing), len(drift.Extra))

	for _, path := range drift.Examples(3) {
		fmt.Fprintln(w, "  docker/"+path)
	}
	if remainder := drift.Total() - 3; remainder > 0 {
		fmt.Fprintf(w, "  and %d more\n", remainder)
	}

	fmt.Fprintln(w, "Rebuild it — go build -o madock . in "+execDir+
		" — unless you are editing templates and know why they differ")
}

// TemplateDrift reports templates on disk that the running binary was not built
// from, and nil when there is nothing to say.
//
// **Only a source checkout can drift.** Everywhere else the two are kept
// together by construction: a downloaded binary extracts its own snapshot and
// stamps it, and madock-pro's installation gets docker/ the same way because the
// assets belong to the imported module. In a clone they are separate objects —
// git delivers the templates, a build delivers the binary — and nothing married
// them.
//
// What that cost, measured 2026-08-23: the live binary was 3.9.17 built 19
// August, the tree was 4.0.1 merged 21 August, and the php snippets had been
// reorganised in between. Every rebuild ended as
//
//	target php: failed to solve: failed to read dockerfile: open php.Dockerfile: no such file or directory
//
// with the containers already destroyed — rebuild takes them down before it
// generates the build context. Nothing said the binary was old: `--version`
// answers for the binary alone, and there was no second version to compare it
// with. The drift was only ever visible as a failed rebuild, which is after the
// damage.
//
// The comparison is content, not timestamps. `git checkout` rewrites mtimes on
// files it did not change and leaves them on files it restored, so a clock
// answers a question nobody asked; the embedded snapshot is what this binary
// actually knows, so comparing it against the disk answers exactly "was this
// binary built from these templates".
func TemplateDrift() *Drift {
	if DockerFS == nil {
		return nil
	}

	execDir := paths.GetExecDirPath()
	if !isSourceCheckout(execDir) {
		return nil
	}

	root := filepath.Join(execDir, "docker")
	drift := &Drift{}
	inBinary := map[string]bool{}

	fs.WalkDir(DockerFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "." || d.IsDir() {
			return err
		}

		inBinary[path] = true

		embeddedBody, err := fs.ReadFile(DockerFS, path)
		if err != nil {
			return nil
		}

		onDisk, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			drift.Missing = append(drift.Missing, path)
			return nil
		}

		if !bytes.Equal(embeddedBody, onDisk) {
			drift.Changed = append(drift.Changed, path)
		}

		return nil
	})

	collectExtra(root, inBinary, drift)

	sort.Strings(drift.Changed)
	sort.Strings(drift.Missing)
	sort.Strings(drift.Extra)

	if drift.Total() == 0 {
		return nil
	}

	return drift
}

// collectExtra finds files on disk that the binary has never heard of — the half
// of the drift the embedded walk cannot see, and the half a reorganised template
// tree shows up as.
//
// It descends only into the top-level directories the embed itself contains,
// which is what keeps it from reporting docker/embed.go and the tests beside it:
// the //go:embed patterns name asset directories, so everything at the root of
// docker/ is source and belongs to no snapshot.
func collectExtra(root string, inBinary map[string]bool, drift *Drift) {
	tops, err := fs.ReadDir(DockerFS, ".")
	if err != nil {
		return
	}

	for _, top := range tops {
		if !top.IsDir() {
			continue
		}

		filepath.WalkDir(filepath.Join(root, top.Name()), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			// macOS writes these into any directory that has been opened in
			// Finder. They are in no embed and never will be, so reporting them
			// would make the check cry wolf on every machine it runs on.
			if d.Name() == ".DS_Store" {
				return nil
			}

			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}

			if slash := filepath.ToSlash(rel); !inBinary[slash] {
				drift.Extra = append(drift.Extra, slash)
			}

			return nil
		})
	}
}
