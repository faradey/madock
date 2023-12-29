package open

import (
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"log"
	"os/exec"
	"runtime"
)

type ArgsStruct struct {
	attr.Arguments
}

func Execute() {
	attr.Parse(new(ArgsStruct))

	projectConfig := configs.GetCurrentProjectConfig()
	hosts := configs.GetHosts(projectConfig)
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, "https://"+hosts[0]["name"])
	err := exec.Command(cmd, args...).Start()
	if err != nil {
		log.Fatal(err)
	}
}
