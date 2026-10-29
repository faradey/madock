package prune

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
		Aliases:    []string{"system:prune"},
		JSONOutput: true,
		Handler:    Execute,
		Help:       "Free Docker disk: --images (unused images of madock projects), --build-cache (the machine's build cache). Shows what it would free; removes only with --yes. Never touches volumes. Supports --json (-j)",
		Category:   "general",
		ArgsType:   new(arg_struct.ControllerGeneralSystemPrune),
		Global:     true,
	})
}

// Result is the --json answer.
type Result struct {
	Plan            docker.PrunePlan `json:"plan"`
	Removed         bool             `json:"removed"`
	ImagesRemoved   int              `json:"images_removed"`
	ImagesRefused   []string         `json:"images_refused,omitempty"`
	BuildCacheFreed bool             `json:"build_cache_removed"`
}

// Execute shows, and with --yes removes, what system:prune may remove.
//
// Behind allow_destructive_commands although neither kind is data. On a
// server the build cache is what keeps a deploy's rebuild to a recreate: without
// it the next deploy builds every image from nothing, from the network, and on
// 2026-10-22 one bad answer from pecl.php.net failed exactly such a build on a
// production rebuild. A server, where the switch is off as madock Pro ships it,
// is where that must not happen by habit. Showing what would go needs no switch.
func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralSystemPrune)).(*arg_struct.ControllerGeneralSystemPrune)
	var w *output.Writer
	if args.Json {
		w = output.Begin()
	}
	fail := func(msg string) {
		if w != nil {
			w.Fail("%s", msg)
		}
		fmtc.ErrorLn(msg)
		os.Exit(1)
	}

	if !args.Images && !args.BuildCache {
		fail("Name what to free: --images, --build-cache, or both. Volumes are never removed here; that is project:remove")
	}

	plan, err := docker.PlanPrune()
	if err != nil {
		fail("Cannot read what Docker holds: " + err.Error())
	}
	if !args.Images {
		plan.Images, plan.ImageBytes = nil, 0
	}
	if !args.BuildCache {
		plan.BuildCacheBytes = 0
	}

	if !args.Yes {
		if w != nil {
			w.Print(Result{Plan: plan})
			return
		}
		show(plan, args)
		fmtc.ToDoLn("Nothing was removed. Run again with --yes to remove it")
		return
	}

	if !configs.AllowsDestructiveCommands() {
		refusal := configs.DestructiveRefusal("system:prune")
		refusal[0] = "'system:prune --yes' removes images and build cache, and this installation does not allow destructive commands."
		if w != nil {
			w.Fail("%s", refusal[0])
		}
		for _, line := range refusal {
			fmtc.ErrorLn(line)
		}
		os.Exit(1)
	}

	result := Result{Plan: plan, Removed: true}
	if args.Images {
		result.ImagesRemoved, result.ImagesRefused = docker.RemoveImages(plan.Images)
	}
	if args.BuildCache {
		if err := docker.PruneBuildCache(); err != nil {
			if w != nil {
				w.FailWith(result, "%s", err.Error())
			}
			fmtc.ErrorLn(err.Error())
			os.Exit(1)
		}
		result.BuildCacheFreed = true
	}

	if w != nil {
		w.Print(result)
		return
	}
	if args.Images {
		fmtc.SuccessLn(fmt.Sprintf("Removed %d of %d images (%s)", result.ImagesRemoved, len(plan.Images), docker.HumanBytes(plan.ImageBytes)))
		for _, line := range result.ImagesRefused {
			fmtc.WarningLn("  kept: " + line)
		}
	}
	if args.BuildCache {
		fmtc.SuccessLn("Removed the build cache (" + docker.HumanBytes(plan.BuildCacheBytes) + ")")
	}
}

func show(plan docker.PrunePlan, args *arg_struct.ControllerGeneralSystemPrune) {
	if args.Images {
		fmtc.TitleLn(fmt.Sprintf("Images of madock projects no container uses: %d, %s", len(plan.Images), docker.HumanBytes(plan.ImageBytes)))
		for _, img := range plan.Images {
			fmt.Printf("  %-12s %10s  %s\n", img.ID, docker.HumanBytes(img.Bytes), img.Name)
		}
		fmt.Println("  The next start of each project pulls or builds them again.")
	}
	if args.BuildCache {
		fmtc.TitleLn("Build cache not in use: " + docker.HumanBytes(plan.BuildCacheBytes))
		fmt.Println("  This is the machine's, not only madock's: every build on it starts from nothing afterwards.")
	}
}
