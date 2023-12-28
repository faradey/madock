package open

import (
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"log"
	"os"
	"os/exec"
	"runtime"
)

type ArgsStruct struct {
	attr.Arguments
}

func Execute() {
	getArgs()

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

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if attr.IsParseArgs && len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		p, err := arg.NewParser(arg.Config{
			IgnoreEnv: true,
		}, args)

		if err != nil {
			log.Fatal(err)
		}

		err = p.Parse(argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	attr.IsParseArgs = false
	return args
}
