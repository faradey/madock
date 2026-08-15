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
// This is the maintainer's form, for the tree madock ships. A user with an
// override of their own has `madock template:convert`, which needs no Go
// toolchain; both call tmpl.ConvertTree, so neither can drift from the
// conversion the renderer applies at read time.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/faradey/madock/v3/src/helper/tmpl"
)

func main() {
	dir := flag.String("dir", "docker", "directory of templates to convert")
	dryRun := flag.Bool("dry-run", false, "report what would change without writing")
	flag.Parse()

	report, err := tmpl.ConvertTree(*dir, *dryRun)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Print(report.String())
}
