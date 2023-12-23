package media

import (
	"fmt"
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/controller/general/remote_sync"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/finder"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/pkg/sftp"
	"log"
	"os"
	"sync"
	"time"
)

type ArgsStruct struct {
	attr.Arguments
	ImagesOnly bool `arg:"-i,--images-only" help:"Sync images only"`
	Compress   bool `arg:"-c,--compress" help:"Compress images"`
}

func Execute() {
	args := getArgs()

	projectConf := configs.GetCurrentProjectConfig()
	remoteDir := projectConf["SSH_SITE_ROOT_PATH"]
	maxProcs := finder.MaxParallelism() - 1
	var scTemp *sftp.Client
	isFirstConnect := false
	paths.MakeDirsByPath(paths.GetRunDirPath() + "/pub/media")
	for maxProcs > 0 {
		conn := remote_sync.Connect(projectConf["SSH_AUTH_TYPE"], projectConf["SSH_KEY_PATH"], projectConf["SSH_PASSWORD"], projectConf["SSH_HOST"], projectConf["SSH_PORT"], projectConf["SSH_USERNAME"])
		if !isFirstConnect {
			fmt.Println("")
			fmt.Println("Server connection...")
			isFirstConnect = true
		}
		defer remote_sync.Disconnect(conn)
		scTemp = remote_sync.NewClient(conn)
		defer scTemp.Close()
		maxProcs--
	}

	fmt.Println("\n" + "Synchronization is started")
	ch := make(chan bool, 15)
	var chDownload sync.WaitGroup
	go remote_sync.ListFiles(&chDownload, ch, remoteDir+"/pub/media/", "", 0, args.ImagesOnly, args.Compress)
	time.Sleep(3 * time.Second)
	chDownload.Wait()
	fmt.Println("\n" + "Synchronization is completed")
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

	return args
}
