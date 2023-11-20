package remote_sync_db

import (
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/ssh"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
	Name string `long:"name" short:"n" description:"Name of the archive file"`
}

func Execute() {
	// TODO add CLI args
	args := getArgs()

	projectConf := configs.GetCurrentProjectConfig()
	conn := ssh.Connect(projectConf["SSH_AUTH_TYPE"], projectConf["SSH_KEY_PATH"], projectConf["SSH_PASSWORD"], projectConf["SSH_HOST"], projectConf["SSH_PORT"], projectConf["SSH_USERNAME"])
	ssh.DbDump(conn, projectConf["SSH_SITE_ROOT_PATH"], args.Name)
}

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		var err error
		_, err = flags.ParseArgs(args, argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	return args
}
