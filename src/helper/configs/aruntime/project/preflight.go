package project

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/faradey/madock/v3/src/helper/tmpl"
)

// BrokenIncludes lists the project's own templates whose includes no longer
// resolve, without rendering or writing anything.
//
// This is a preflight for the commands that destroy before they render. Both
// `rebuild` and `restart` stop the containers first and generate the build
// context second, so a template that cannot be assembled used to end the process
// with the environment already down — and the message, "The file
// snippets/dockerfile/php/nodejs does not exist", named a path rather than a
// cause. **Measured on 2026-08-18**: a production machine and then the demo
// machine were taken down that way, by `rebuild` in a maintenance window and by
// an ordinary `restart` afterwards.
//
// The drift it catches is structural rather than accidental. A project's own
// templates under `.madock/docker/` survive every madock upgrade, while the
// snippets they include ship inside the binary and move between releases —
// `php/nodejs` became `common/nodejs`. Nothing reconciles the two, and nothing
// looks until a build.
//
// Only the project's own copies are walked, and only missing includes are
// reported. A template that fails to parse is left to the render, which says so
// properly: a preflight that can fail for reasons unrelated to what it guards is
// one people learn to pass with --force.
func BrokenIncludes(projectName string) []string {
	checker := &tmpl.Renderer{
		Snippet: func(name string) (string, error) {
			file, err := FindSnippetFile(projectName, name)
			if err != nil {
				return "", err
			}
			body, err := os.ReadFile(file)
			return string(body), err
		},
		// Silent on purpose: the legacy-syntax warning belongs to the render that
		// converts the file, and printing it from a check as well would say it
		// twice for every template.
		OnLegacy: nil,
	}

	var problems []string
	for _, root := range projectOwnedTemplateDirs(projectName) {
		problems = append(problems, brokenIn(root, checker)...)
	}
	return problems
}

// ReportBrokenIncludes prints what BrokenIncludes found and reports whether the
// caller should stop.
//
// One text for every command that has to make this check, because the reader's
// next question is always the same — which file, and what do I do about it — and
// answering it differently in each place is how half of them end up not
// answering it at all.
func ReportBrokenIncludes(projectName string) bool {
	problems := BrokenIncludes(projectName)
	if len(problems) == 0 {
		return false
	}

	fmtc.ErrorLn("This project's templates include files that do not exist, so the build context cannot be generated:")
	for _, problem := range problems {
		fmtc.WarningLn("  " + problem)
	}
	fmtc.ToDoLn("These are the project's own templates, and they survive madock upgrades while the")
	fmtc.ToDoLn("snippets they include ship inside the binary and move between releases. Point the")
	fmtc.ToDoLn("include at the new path, or delete the override to fall back to what madock ships.")
	fmtc.WarningLn("Nothing was stopped or rebuilt.")

	return true
}

// projectOwnedTemplateDirs are the two places a project's own templates live:
// the copy in its repository, and the machine's copy for this project. What the
// installation ships is not walked — it moves with the binary, and its own tests
// cover it.
func projectOwnedTemplateDirs(projectName string) []string {
	var dirs []string
	for _, dir := range []string{
		paths.GetRunDirPath() + "/.madock/docker",
		paths.GetExecDirPath() + "/projects/" + projectName + "/docker",
	} {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

func brokenIn(root string, checker *tmpl.Renderer) []string {
	var problems []string

	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil //nolint:nilerr // an unreadable file is the render's problem to report
		}

		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if !strings.Contains(string(body), tmpl.LeftDelim) && !tmpl.IsLegacy(string(body)) {
			return nil // no directives at all, so nothing to include
		}

		if checkErr := checker.Check(shortName(root, path), string(body)); checkErr != nil {
			if errors.Is(checkErr, ErrSnippetMissing) {
				problems = append(problems, fmt.Sprintf("%s\n  %s", path, indent(checkErr.Error())))
			}
		}
		return nil
	})

	return problems
}

// shortName is what the template is called in an error — the path relative to
// the directory it was found in, which is how includes are written.
func shortName(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return rel
	}
	return path
}

func indent(text string) string {
	return strings.ReplaceAll(strings.TrimSpace(text), "\n", "\n  ")
}
