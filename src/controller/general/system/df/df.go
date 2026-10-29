package df

import (
	"fmt"
	"os"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/cli/output"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
)

func init() {
	command.Register(&command.Definition{
		Aliases:    []string{"system:df"},
		JSONOutput: true,
		Handler:    Execute,
		Help:       "Show what fills the Docker disk: images, volumes and build cache, in total and per project. Removes nothing. Supports --json (-j) output",
		Category:   "general",
		// Global: the disk is the machine's, and the question is usually asked
		// from wherever one stands when it fills up.
		Global: true,
	})
}

// Execute prints the Docker disk by kind and by project.
//
// **Read-only, and deliberately so**, for the reason project:orphans is: what
// it reports includes volumes, which are databases, and a removal here would be
// a second route around `allow_destructive_commands`. It says what each kind
// is, so the decision about what may go is made on numbers.
func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralSystemDf)).(*arg_struct.ControllerGeneralSystemDf)

	known := map[string]bool{}
	for _, entry := range configs.ListProjects() {
		known[entry.Name] = true
	}

	usage, err := docker.ReadDiskUsage(known)
	if err != nil {
		if args.Json {
			output.PrintJSONError("Cannot read the Docker disk usage: " + err.Error())
			os.Exit(1)
		}
		fmtc.ErrorLn("Cannot read the Docker disk usage: " + err.Error())
		os.Exit(1)
	}

	if args.Json {
		output.PrintJSON(usage)
		return
	}

	fmtc.TitleLn("Docker disk")
	row := func(name string, k docker.KindUsage, what string) {
		fmt.Printf("  %-16s %5d  %10s   unused %10s   %s\n", name, k.Count, docker.HumanBytes(k.Bytes), docker.HumanBytes(k.Reclaimable), what)
	}
	row("build cache", usage.BuildCache, "rebuilt by the next build")
	row("images", usage.Images, "pulled or built again on start")
	row("  of them <none>", usage.Dangling, "untagged leftovers of rebuilds")
	row("volumes", usage.Volumes, "DATA: databases, indexes, queues")

	if len(usage.Projects) == 0 {
		return
	}
	fmt.Println()
	fmtc.TitleLn("By project (images: what is the project's own; volumes: everything)")
	for _, p := range usage.Projects {
		mark := ""
		if !p.Registered {
			mark = "   not registered"
		}
		fmt.Printf("  %-32s images %3d %10s   volumes %3d %10s%s\n",
			p.Project, p.Images, docker.HumanBytes(p.ImageBytes), p.Volumes, docker.HumanBytes(p.VolumeBytes), mark)
	}
}
