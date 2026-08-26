package cron

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/cli/output"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/paths"
)

type ArgsStruct struct {
	attr.Arguments
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"cron:enable"},
		Handler:  Enable,
		Help:     "Enable cron",
		Category: "cron",
		ArgsType: new(ArgsStruct),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"cron:disable"},
		Handler:  Disable,
		Help:     "Disable cron",
		Category: "cron",
		ArgsType: new(ArgsStruct),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"cron:status"},
		Handler:  Status,
		Help:     "Report whether cron is running and how many jobs are installed. Supports --json (-j). Exit code: 0 healthy, 1 problem, 2 could not tell",
		Category: "cron",
		ArgsType: new(ArgsStruct),
	})
}

func Enable() {
	attr.Parse(new(ArgsStruct))
	projectName := configs.GetProjectName()
	projectConfig := configs.GetProjectConfig(projectName)
	configs.SetParam(projectName, "cron/enabled", "true", projectConfig["activeScope"], "")
	docker.CronExecute(projectName, true, true)
	reportOverriddenSetting(projectName, "true")
}

// reportOverriddenSetting says so when the value just written is not the value
// in effect.
//
// `cron:enable` writes cron/enabled into the installation's copy of the project
// config, and a project's own committed .madock/config.xml is merged over that —
// so on a project whose file says false, the command starts cron, prints "Cron
// was started", and the setting reads false the moment anybody asks. The daemon
// really is running, which is what makes it convincing; the next `start` turns
// it off again, because that is where the effective value is read. Measured on
// extmag-core-bigcommerce on 2026-08-26.
//
// A warning rather than a write: that file belongs to whoever committed it, and
// madock editing it behind their back is the thing .madock/config.xml exists to
// prevent.
func reportOverriddenSetting(projectName, want string) {
	conf := configs.GetProjectConfig(projectName)
	inEffect := strings.ToLower(conf["cron/enabled"])
	if inEffect == want {
		return
	}

	fmtc.WarningLn("cron/enabled is " + inEffect + " for this project, not " + want + " — something is overriding what was just written.")

	if path := conf["path"]; path != "" {
		dir := strings.TrimRight(path, "/") + "/.madock"
		own := dir + "/config.xml"
		if paths.IsFileExist(own) {
			fmtc.WarningLn("  " + own + " is merged over madock's own copy and wins. Edit cron/enabled there.")
			reportReleaseCopy(dir)
		}
	}

	if want == "true" {
		fmtc.WarningLn("  Cron is running now, and the next `start` will read the effective value and stop it again.")
	}
}

// reportReleaseCopy says when the file just named is not a file but a way into
// the current release.
//
// Where deployer manages a project, `<path>/.madock` is a symlink to
// `current/.madock`, so the path above resolves into `releases/<n>/` — a
// directory the next deploy replaces. Editing there works, and lasts until the
// next release, at which point the value comes back from the repository and the
// setting silently reverts. Measured on `extmag` on 2026-08-27: fixing this for
// one project took three edits, and only the one in git survives a deploy.
func reportReleaseCopy(dir string) {
	info, err := os.Lstat(dir)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return
	}

	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		// The link is there and its target cannot be read. Still worth saying,
		// because the half that matters — this is a release, not the source —
		// does not depend on knowing where it points.
		resolved = "the current release"
	}

	fmtc.WarningLn("  That path is a symlink into the current release (" + resolved + ").")
	fmtc.WarningLn("  An edit there holds until the next deploy, which brings the file back from the repository — change it in the project's repository as well.")
}

func Disable() {
	attr.Parse(new(ArgsStruct))
	projectName := configs.GetProjectName()
	projectConfig := configs.GetProjectConfig(projectName)
	configs.SetParam(projectName, "cron/enabled", "false", projectConfig["activeScope"], "")
	docker.CronExecute(projectName, false, true)
	reportOverriddenSetting(projectName, "false")
}

// Report is what `cron:status` answers, in the shape a script reads it.
//
// Three separate questions, kept separate: what the configuration asks for,
// whether a daemon is actually running, and whether it has anything to run.
// Jobs is -1 when the crontab could not be read, which is never rounded down to
// none.
type Report struct {
	Enabled   bool `json:"enabled"`
	Running   bool `json:"running"`
	Jobs      int  `json:"jobs"`
	JobsKnown bool `json:"jobs_known"`
	// State is the verdict the exit code carries: ok, problem or unknown.
	State string `json:"state"`
	// Reason says what is wrong, and is empty when nothing is.
	Reason string `json:"reason,omitempty"`
}

// Status answers "is the scheduler alive" without changing anything.
//
// It exists because there was no way to ask. `cron:enable` and `cron:disable`
// both act, and `status` folds cron in among the containers — so the only
// read-only answer available after a deploy came from a command that reports
// the whole project, and a script had to parse prose or reach into
// `status --json`. A deploy that has just restarted the application's container
// is exactly when this question is worth asking, so it is a command of its own
// and it carries the answer in the exit code:
//
//	0  what the configuration asks for is what the container has
//	1  a problem: cron is enabled and not running, or running with nothing to run
//	2  could not tell — the container did not answer
//
// Two and one are deliberately different. A check that cannot reach the
// container has not established that cron is fine, and rounding that up to
// healthy is how the silence this command exists to break gets restored.
func Status() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

	projectName := configs.GetProjectName()
	projectConf := configs.GetProjectConfig(projectName)

	report := Report{
		Enabled: strings.ToLower(projectConf["cron/enabled"]) == "true",
		Running: docker.CronRunning(projectName),
		Jobs:    -1,
	}
	if report.Running {
		report.Jobs, report.JobsKnown = docker.CronJobCount(projectName)
		if !report.JobsKnown {
			report.Jobs = -1
		}
	}

	switch {
	case report.Running && !report.JobsKnown:
		report.State = "unknown"
		report.Reason = "cron is running, but its crontab could not be read"
	case report.Running && report.Jobs == 0:
		report.State = "problem"
		report.Reason = "cron is running and no jobs are installed — nothing runs on a schedule"
	case report.Running:
		report.State = "ok"
	case report.Enabled:
		report.State = "problem"
		report.Reason = "cron is enabled in the configuration and no cron daemon is running in the container"
	default:
		report.State = "ok"
		report.Reason = "cron is not enabled for this project"
	}

	if args.Json {
		_ = output.PrintJSON(report)
	} else {
		printReport(report)
	}

	switch report.State {
	case "problem":
		os.Exit(1)
	case "unknown":
		os.Exit(2)
	}
}

func printReport(report Report) {
	switch report.State {
	case "ok":
		if !report.Running {
			fmtc.SuccessLn("Cron is not enabled for this project, and no cron daemon is running")
			return
		}
		if report.Jobs == 1 {
			fmtc.SuccessLn("Cron is running (1 job)")
			return
		}
		fmtc.SuccessLn(fmt.Sprintf("Cron is running (%d jobs)", report.Jobs))
	case "problem":
		fmtc.WarningLn(report.Reason)
		if report.Enabled && !report.Running {
			fmt.Println("  Start it with `madock cron:enable`.")
		}
	default:
		fmtc.WarningLn(report.Reason)
	}
}
