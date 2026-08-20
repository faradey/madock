package dockerassets

import (
	"io/fs"
	"testing"
	"text/template"

	"github.com/faradey/madock/v4/src/helper/tmpl"
)

// TestEveryTemplateParses is what the tag-balance test became.
//
// That test walked this tree counting <<<if against <<<endif>>>, because the
// engine that read them could not complain: it located the closing tag by
// counting openings, so one unbalanced tag — including one written inside a
// comment — made it abandon the whole file with every conditional unresolved,
// and the result was written out anyway. That produced an nginx config with six
// server blocks where one belonged, and nothing said a word.
//
// A parser says a word. It names the file and the line, it catches an unknown
// function and a malformed action as well as an unbalanced one, and it is the
// same parse that runs on a user's machine — so a template that fails here
// could never have rendered there.
func TestEveryTemplateParses(t *testing.T) {
	err := fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		body, readErr := fs.ReadFile(FS, path)
		if readErr != nil {
			return readErr
		}

		if _, parseErr := template.New(path).Delims(tmpl.LeftDelim, tmpl.RightDelim).Funcs(tmpl.StubFuncs()).Parse(string(body)); parseErr != nil {
			t.Errorf("%s does not parse: %v", path, parseErr)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded templates: %v", err)
	}
}

// TestNoTemplateIsStillLegacy keeps the old syntax out of the tree.
//
// The renderer converts a template written against <<<if>>> on the fly and
// warns, which is what keeps a project's own override under .madock/docker/
// working across the change. Nothing madock ships should ever take that path:
// a template here that still needs converting means the converter was not run
// after a rebase, and the warning would go to every user of the release.
func TestNoTemplateIsStillLegacy(t *testing.T) {
	err := fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		body, readErr := fs.ReadFile(FS, path)
		if readErr != nil {
			return readErr
		}

		if tmpl.IsLegacy(string(body)) {
			t.Errorf("%s is still in the old syntax — run: go run ./tools/tmplconvert", path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded templates: %v", err)
	}
}
