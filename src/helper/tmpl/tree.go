package tmpl

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// ConvertTree rewrites a directory of templates from the old <<<if>>> syntax.
//
// It lives here rather than in the command that calls it because two callers
// need it and one of them is a user's: `madock template:convert` is the whole
// answer for somebody with an override under .madock/docker/, who has a binary
// and no Go toolchain to run tools/tmplconvert with. Documenting a `go run` for
// them was a hole, and this is what closes it.
func ConvertTree(dir string, dryRun bool) (Report, error) {
	report := Report{Dir: dir}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		// A tree can carry its own Go files — madock's does, for the embed
		// directive and the tests beside it. They are not templates.
		if strings.HasSuffix(path, ".go") {
			return nil
		}

		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		out, notes := Legacy(string(body))
		for _, note := range notes {
			report.Notes = append(report.Notes, path+": "+note)
		}

		// Parsing is the check that the result is a template at all. It costs
		// nothing here and catches a conversion that the engine would otherwise
		// reject at the next `madock start`.
		if err := Parses(path, out); err != nil {
			report.Notes = append(report.Notes, err.Error())
		}

		if out == string(body) {
			report.Unchanged++
			return nil
		}
		report.Converted = append(report.Converted, path)

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
		return report, err
	}

	sort.Strings(report.Notes)
	return report, nil
}

// Report is what a conversion did, in the shape both callers print.
type Report struct {
	Dir       string
	Converted []string
	Unchanged int
	Notes     []string
}

func (r Report) String() string {
	var out strings.Builder

	fmt.Fprintf(&out, "%d converted, %d already in the current syntax, under %s\n", len(r.Converted), r.Unchanged, r.Dir)
	for _, path := range r.Converted {
		fmt.Fprintf(&out, "  %s\n", path)
	}

	if len(r.Notes) > 0 {
		fmt.Fprintf(&out, "\n%d things to look at:\n", len(r.Notes))
		for _, note := range r.Notes {
			fmt.Fprintf(&out, "  %s\n", note)
		}
	}

	return out.String()
}

// Parses answers whether text is a template at all, without executing it.
func Parses(name, body string) error {
	_, err := template.New(name).Delims(LeftDelim, RightDelim).Funcs(StubFuncs()).Parse(body)
	return err
}
