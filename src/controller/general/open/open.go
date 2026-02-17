package open

import (
	"os/exec"
	"runtime"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"open"},
		Handler:  Execute,
		Help:     "Open project in browser",
		Category: "general",
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralOpen)).(*arg_struct.ControllerGeneralOpen)

	projectConfig := configs.GetCurrentProjectConfig()
	hosts := configs.GetHosts(projectConfig)
	var cmd string
	var argsCommand []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		argsCommand = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	host := "https://" + hosts[0]["name"]
	if args.Service != "" {
		host = host + "/" + args.Service
	}
	argsCommand = append(argsCommand, host)
	err := exec.Command(cmd, argsCommand...).Start()
	if err != nil {
		logger.Fatal(err)
	}
}
