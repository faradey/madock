package template

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/faradey/madock/v3/src/helper/tmpl"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"template:convert"},
		Handler:  Convert,
		Help:     "Rewrite docker template overrides from the old <<<if>>> syntax to text/template",
		Category: "config",
		ArgsType: new(arg_struct.ControllerGeneralTemplateConvert),
		// Global: a directory of templates is a directory of templates. Given a
		// path it needs no project at all, and refusing to run outside one would
		// make the command useless in exactly the place somebody keeps a copy of
		// their overrides.
		Global: true,
	})
}

// Convert rewrites a directory of templates in place.
//
// It exists because the conversion had to be reachable by whoever needs it. A
// project's own overrides under .madock/docker/ are converted as they are read,
// with a warning, so nothing breaks — but the warning has to be able to say what
// to run, and `go run ./tools/tmplconvert` is only an answer for somebody who
// installed from source. Everyone with a downloaded binary was being told to
// use a path they do not have.
func Convert() {
	args := attr.Parse(new(arg_struct.ControllerGeneralTemplateConvert)).(*arg_struct.ControllerGeneralTemplateConvert)

	dir := args.Path
	if dir == "" {
		dir = filepath.Join(paths.GetRunDirPath(), ".madock", "docker")
		if !paths.IsFileExist(dir) {
			logger.Fatal(fmt.Errorf("no templates to convert: %s does not exist. Pass a directory to convert one somewhere else", dir))
		}
	}

	info, err := os.Stat(dir)
	if err != nil {
		logger.Fatal(err)
	}
	if !info.IsDir() {
		logger.Fatal(fmt.Errorf("%s is a file; this converts a directory of templates", dir))
	}

	report, err := tmpl.ConvertTree(dir, args.DryRun)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Print(report.String())

	if args.DryRun && len(report.Converted) > 0 {
		fmtc.ToDoLn("Nothing was written. Run the same command without --dry-run to convert.")
	}
}
