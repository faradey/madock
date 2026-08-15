// Command tmplconvert rewrites madock's docker templates from the hand-written
// <<<if>>> DSL into text/template.
//
// It exists because the rewrite touches every template there is, and a hand
// edit of 276 files cannot be rebased. With a converter, a branch that changed
// a template is landed by taking theirs, running this again and re-running the
// golden tests — one command instead of 276 conflicts. That is the reason it is
// committed rather than run once and thrown away.
//
//	go run ./tools/tmplconvert            # rewrite docker/ in place
//	go run ./tools/tmplconvert -dry-run   # say what would change, touch nothing
//	go run ./tools/tmplconvert -dir path  # somewhere else
//
// The conversion itself lives in src/helper/tmpl, because the renderer applies
// it to a project's own overrides at render time as well: a file in
// <PROJECT>/.madock/docker/ written against the old syntax keeps working, with
// a warning, instead of needing a second engine kept alive to read it.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/faradey/madock/v3/src/helper/tmpl"
)

func main() {
	dir := flag.String("dir", "docker", "directory of templates to convert")
	dryRun := flag.Bool("dry-run", false, "report what would change without writing")
	flag.Parse()

	report, err := run(*dir, *dryRun)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(report)
}

func run(dir string, dryRun bool) (string, error) {
	var converted, unchanged int
	var notes []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		// The tree carries its own Go files — the embed directive and the test
		// that counts tags. They are not templates.
		if strings.HasSuffix(path, ".go") {
			return nil
		}

		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		out, fileNotes := tmpl.Legacy(string(body))
		for _, note := range fileNotes {
			notes = append(notes, path+": "+note)
		}

		// Parsing is the check that the result is a template at all. It costs
		// nothing here and catches a conversion that produced something the
		// engine would only reject on a user's machine, at `madock start`.
		if err := parses(path, out); err != nil {
			notes = append(notes, err.Error())
		}

		if out == string(body) {
			unchanged++
			return nil
		}
		converted++
		if dryRun {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(path, []byte(out), info.Mode().Perm())
	})
	if err != nil {
		return "", err
	}

	var report strings.Builder
	verb := "converted"
	if dryRun {
		verb = "would convert"
	}
	fmt.Fprintf(&report, "%s %d files, %d already plain text\n", verb, converted, unchanged)
	if len(notes) > 0 {
		sort.Strings(notes)
		fmt.Fprintf(&report, "\n%d things to look at:\n", len(notes))
		for _, note := range notes {
			fmt.Fprintf(&report, "  %s\n", note)
		}
	}
	return report.String(), nil
}

// parses answers whether the converted text is a template at all.
func parses(name, body string) error {
	_, err := template.New(name).Delims(tmpl.LeftDelim, tmpl.RightDelim).Funcs(tmpl.StubFuncs()).Parse(body)
	return err
}
