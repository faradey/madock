//go:build e2e

package e2e

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// templateMarkers are the two things that must never survive rendering.
//
// `{{{` opens every action in the current engine, and `<<<if` opened a
// condition in the DSL it replaced. Either of them in a generated file means
// the renderer walked past something it did not understand and copied it out
// verbatim — docker then reads a compose file, an nginx config or a Dockerfile
// with template syntax in it, and what it says about that never mentions
// madock.
//
// The closing delimiters are deliberately not looked for. `}}}` occurs in real
// content — three nested braces at the end of a JSON or JS fragment — and a
// check that cries wolf on a valid file is a check people delete.
var templateMarkers = []string{"{{{", "<<<if"}

// TestNoTemplatePlaceholderSurvivesRendering sweeps everything a project
// generates for template syntax that was copied out instead of evaluated.
//
// It exists because of a failure with no error message. A binary older than the
// templates unpacked beside it does not know a key a newer snippet uses, so the
// snippet is written to disk with its braces intact — and the shared proxy
// compose file is one file for every project on the machine, so every project
// on it stops at once. What that looks like from outside is docker refusing a
// file nobody edited, on a machine where nothing changed.
//
// The sweep is the whole assertion. Naming the files that may contain
// placeholders would be a list to maintain, and the interesting case is always
// the file nobody thought of.
func TestNoTemplatePlaceholderSurvivesRendering(t *testing.T) {
	p := newProject(t, "e2etemplate")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2etemplate.test",
	)
	p.run(20*time.Minute, "start")

	// The check runs from the harness after every start, so this is not the only
	// place it happens — but a test of its own is what says the rule exists, and
	// it is the one that fails by name when a placeholder gets out.
	if leftovers := unrenderedTemplates(t, p.install.dir); len(leftovers) > 0 {
		t.Errorf("template syntax survived rendering:\n%s", strings.Join(leftovers, "\n"))
	}

	// And again after a rebuild, which regenerates everything from the
	// templates rather than reusing what start wrote.
	p.run(25*time.Minute, "rebuild")

	if leftovers := unrenderedTemplates(t, p.install.dir); len(leftovers) > 0 {
		t.Errorf("template syntax survived a rebuild:\n%s", strings.Join(leftovers, "\n"))
	}
}

// requireRendered is the same sweep, run from the harness after any command
// that regenerates the runtime.
//
// Every platform test pays for it and none of them had to be written for it:
// Magento, Shopware and Medusa render snippets a `custom` project never
// touches, and those are exactly the ones a change to the engine is most likely
// to leave behind. It reads files that were just written, so it costs
// milliseconds against a command that took minutes.
func (p *project) requireRendered() {
	p.t.Helper()

	if !p.configured() {
		return
	}
	if leftovers := unrenderedTemplates(p.t, p.install.dir); len(leftovers) > 0 {
		p.t.Errorf("template syntax survived rendering:\n%s", strings.Join(leftovers, "\n"))
	}
}

// unrenderedTemplates walks the generated runtime and reports every line still
// carrying template syntax, as `path:line: text`.
//
// The line rather than the file, because a compose file is hundreds of lines
// and "there is a placeholder somewhere in it" starts the search over.
func unrenderedTemplates(t *testing.T, installDir string) []string {
	t.Helper()

	root := filepath.Join(installDir, "aruntime")
	var found []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			// A file that vanished under the walk is not evidence of anything;
			// a directory that cannot be read is worth saying out loud.
			t.Logf("walking %s: %v", path, err)
			return nil
		}
		// Regular files only. `aruntime` holds symlinks into the project — src,
		// ssh, composer and ctx/scripts among them — and a symlink to a
		// directory is neither IsDir() nor readable: the first run logged four
		// "is a directory" lines per render for exactly that. Following them
		// would also take the sweep out of the generated tree and into the
		// project's own source, which no template wrote.
		if !entry.Type().IsRegular() {
			return nil
		}

		info, err := entry.Info()
		if err != nil || info.Size() > maxSweptFile {
			return nil
		}

		body, err := os.ReadFile(path)
		if err != nil {
			t.Logf("reading %s: %v", path, err)
			return nil
		}
		// Certificates and keys live here too. A NUL byte is the cheap way to
		// tell a file that is not text, and text is the only thing a template
		// ever produces.
		if bytes.IndexByte(body, 0) >= 0 {
			return nil
		}

		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			relative = path
		}
		for number, line := range strings.Split(string(body), "\n") {
			for _, marker := range templateMarkers {
				if strings.Contains(line, marker) {
					found = append(found, relative+":"+strconv.Itoa(number+1)+": "+strings.TrimSpace(line))
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Logf("could not walk %s: %v", root, err)
	}

	return found
}

// maxSweptFile keeps the sweep off anything that is not a rendered
// configuration. Nothing a template writes comes close to it.
const maxSweptFile = 4 << 20
