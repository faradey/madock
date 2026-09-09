package logs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logrotate"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"logs:rotate"},
		Handler:  RotateExecute,
		Help:     "Rotate the persisted log files of this project's services",
		Category: "general",
	})
}

// rotatingServices are the ones that mount the log volume. Named rather than
// discovered: asking docker which containers mount a volume costs a call per
// service and answers the same thing every time, and a service added to the
// list here is the same edit as adding the mount.
var rotatingServices = []string{"nginx", "db"}

// RotateExecute is `madock logs:rotate`, and it is also what `start` calls.
//
// It is a command of its own because rotation on start alone is rotation that
// never happens on a machine whose projects are started once and left running —
// which is every server. A person or a cron entry can ask for it directly.
func RotateExecute() {
	projectName := configs.GetProjectName()
	projectConf := configs.GetProjectConfig(projectName)

	for _, line := range RotateProject(projectName, projectConf) {
		fmt.Println(line)
	}
}

// RotateProject rotates every service that keeps files, and reports one line per
// service that did something. It never fails the caller: rotation runs after a
// successful start, and a project whose web server is up must not be reported as
// broken because a log could not be copied.
func RotateProject(projectName string, projectConf map[string]string) []string {
	if projectConf["logs/persist/enabled"] != "true" {
		return nil
	}

	maxSize, err := logrotate.ParseSize(projectConf["logs/persist/max_size"])
	if err != nil {
		fmtc.WarningLn("logs/persist/max_size: " + err.Error() + " — logs were not rotated")
		return nil
	}

	keep, err := strconv.Atoi(projectConf["logs/persist/keep"])
	if err != nil || keep < 1 {
		fmtc.WarningLn("logs/persist/keep is not a number of files — logs were not rotated")
		return nil
	}

	plan := logrotate.Plan{MaxSize: maxSize, Keep: keep}

	var reported []string
	for _, service := range rotatingServices {
		container := docker.GetContainerName(projectConf, projectName, service)
		out, err := logrotate.Rotate(container, plan)
		if err != nil {
			// A container that is not running is the ordinary case — a project
			// with no database, a service switched off — and it is not worth a
			// word. Anything else is, because a rotation that never runs is a
			// disk that fills silently, which is what this exists to prevent.
			if !containerAbsent(out) {
				fmtc.WarningLn("could not rotate the logs of " + service + ": " + err.Error())
			}
			continue
		}
		if out != "" {
			reported = append(reported, service+": "+out)
		}
	}

	return reported
}

// containerAbsent recognises docker saying the container is not there, which is
// the one failure that means nothing is wrong.
func containerAbsent(out string) bool {
	for _, sign := range []string{"No such container", "is not running"} {
		if strings.Contains(out, sign) {
			return true
		}
	}

	return false
}
