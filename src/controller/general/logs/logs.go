package logs

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/platform"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/logrotate"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"logs"},
		Handler:  Execute,
		Help:     "Show container logs",
		Category: "general",
		ArgsType: new(arg_struct.ControllerGeneralLogs),
	})
}

// chooseService settles which container's log to show.
//
// Two ways of naming the same thing, so the only case worth a word is naming it
// twice differently: showing one of them would be a guess, and a guess here
// reads as logs — the reader concludes the other service is quiet.
func chooseService(positional, flag, fallback string) (string, error) {
	if positional != "" && flag != "" && positional != flag {
		return "", fmt.Errorf("two services asked for at once: %q and --service %q. Name one", positional, flag)
	}

	if flag != "" {
		return flag, nil
	}
	if positional != "" {
		return positional, nil
	}

	return fallback, nil
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralLogs)).(*arg_struct.ControllerGeneralLogs)

	projectConf := configs.GetCurrentProjectConfig()

	service, err := chooseService(args.Name, args.Service, platform.GetMainService(projectConf))
	if err != nil {
		fmtc.ErrorLn(err.Error())
		os.Exit(1)
	}

	projectName := configs.GetProjectName()
	container := docker.GetContainerName(projectConf, projectName, service)

	// --file answers the question the stream cannot: what happened before the
	// last rebuild. Docker keeps a container's output inside the container's own
	// directory and deletes it with the container, which on a production machine
	// meant three deploys erased the HTTP record of an intrusion. The services
	// also write files into a volume, and this reads those.
	if args.File {
		showPersisted(container, args.Tail)

		return
	}

	cmd := exec.Command("docker", "logs", container)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Fatal(err)
	}
}

// showPersisted prints the tail of every persisted log the service has.
//
// One `sh -c` inside the container rather than a copy out to the host: the files
// live in a volume, and the only thing guaranteed to be able to read a volume is
// something that mounts it. Missing files are not an error — a project that has
// only just started has written nothing yet, and saying "no such file" about
// that would read as a fault.
func showPersisted(container, tail string) {
	if tail == "" {
		tail = "200"
	}

	script := "cd " + logrotate.LogDir + " 2>/dev/null || { echo 'no persisted logs for this service yet'; exit 0; }; " +
		"found=0; for f in *.log*; do [ -f \"$f\" ] || continue; found=1; " +
		"echo \"--- $f ---\"; tail -n " + tail + " \"$f\"; done; " +
		"[ \"$found\" = 1 ] || echo 'no persisted logs for this service yet'"

	cmd := exec.Command("docker", "exec", container, "sh", "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmtc.ErrorLn("Could not read the persisted logs of " + container + ": " + err.Error())
		fmtc.ToDoLn("The container has to be running: this reads a volume, and a volume is readable only from inside.")
	}
}
