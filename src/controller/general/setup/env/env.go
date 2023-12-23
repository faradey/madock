package env

import (
	"encoding/json"
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
	"os/exec"
)

type ArgsStruct struct {
	attr.Arguments
	Force bool   `arg:"-f,--force" help:"Force"`
	Host  string `arg:"-h,--host" help:"Host"`
}

func Execute() {
	args := getArgs()

	envFile := paths.GetRunDirPath() + "/app/etc/env.php"
	if paths.IsFileExist(envFile) && !args.Force {
		log.Fatal("The env.php file is already exist.")
	} else {
		data, err := json.Marshal(configs.GetCurrentProjectConfig())
		if err != nil {
			log.Fatal(err)
		}

		conf := string(data)
		host := args.Host
		projectName := configs.GetProjectName()
		projectConf := configs.GetCurrentProjectConfig()
		service, user, _ := cli.GetEnvForUserServiceWorkdir("php", "www-data", "")
		cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, service), "php", "/var/www/scripts/php/env-create.php", conf, host)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
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
