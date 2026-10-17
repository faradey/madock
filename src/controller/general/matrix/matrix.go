package matrix

import (
	"fmt"
	"os"
	"strings"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/output"
	"github.com/faradey/madock/v4/src/model/versions"
)

func init() {
	command.Register(&command.Definition{
		Aliases:    []string{"matrix"},
		JSONOutput: true,
		Handler:    Execute,
		Help:       "Show the compatibility matrix: every rule of every platform, the presets and common.xml. Supports --json (-j) and --platform",
		Category:   "general",
		ArgsType:   new(arg_struct.ControllerGeneralMatrix),
		// Global: answers about the binary, not a project.
		Global: true,
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralMatrix)).(*arg_struct.ControllerGeneralMatrix)

	all := versions.ExportMatrix()
	platform := strings.TrimSpace(args.Platform)
	if platform != "" {
		p, ok := versions.ExportPlatform(platform)
		if !ok {
			msg := fmt.Sprintf("no matrix for platform %q; known: %s", platform, strings.Join(versions.MatrixPlatforms(), ", "))
			if args.Json {
				output.PrintJSONError(msg)
			} else {
				fmt.Fprintln(os.Stderr, msg)
			}
			os.Exit(1)
		}
		all.Platforms = []versions.PlatformMatrix{p}
	}

	if args.Json {
		output.PrintJSON(all)
		return
	}

	for _, p := range all.Platforms {
		fmt.Println(p.Name)
		printTools(p.Tools)
		for _, pr := range p.Presets {
			fmt.Printf("  preset %s — %s: %s\n", pr.Version, pr.Name, pr.Description)
		}
		for _, k := range versions.SortedBudgetKeys(p.Budgets) {
			fmt.Printf("  budget %s = %s\n", k, p.Budgets[k])
		}
		fmt.Println()
	}
	fmt.Println("common")
	printTools(all.Common.Tools)
	for _, o := range all.Common.Offers {
		fmt.Printf("  offer %s: %s\n", o.Name, strings.Join(o.Versions, ", "))
	}
}

func printTools(tools []versions.ToolMatrix) {
	for _, t := range tools {
		head := "  " + t.Name
		if t.By != "" {
			head += " (by " + t.By + ")"
		}
		fmt.Println(head)
		for _, r := range t.Rules {
			if r.From != "" {
				fmt.Printf("    from %s → %s\n", r.From, shown(r.Value))
			} else {
				fmt.Printf("    = %s → %s\n", r.Version, shown(r.Value))
			}
		}
		if t.Default != nil {
			fmt.Printf("    default → %s\n", shown(*t.Default))
		}
	}
}

// shown makes an empty answer visible: a rule that gives nothing is a rule.
func shown(v string) string {
	if v == "" {
		return "(none)"
	}
	return v
}
