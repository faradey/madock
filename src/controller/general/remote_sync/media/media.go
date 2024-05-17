package media

import (
	"fmt"
	"github.com/faradey/madock/src/controller/general/remote_sync"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/finder"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/pkg/sftp"
	"sync"
	"time"
)

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralRemoteSyncMedia)).(*arg_struct.ControllerGeneralRemoteSyncMedia)

	projectConf := configs.GetCurrentProjectConfig()

	maxProcs := finder.MaxParallelism() - 1
	var scTemp *sftp.Client
	isFirstConnect := false
	paths.MakeDirsByPath(paths.GetRunDirPath() + "/pub/media")
	sshType := "ssh"
	if args.SshType != "" {
		sshType += "_" + args.SshType
	}
	siteRootPath := projectConf[sshType+"/site_root_path"]
	if _, ok := projectConf[sshType+"/site_root_path"]; !ok {
		siteRootPath = projectConf["ssh/site_root_path"]
	}
	for maxProcs > 0 {
		conn := remote_sync.Connect(projectConf, sshType)
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
	go remote_sync.ListFiles(&chDownload, ch, siteRootPath+"/pub/media/", "", 0, args.ImagesOnly, args.Compress)
	time.Sleep(3 * time.Second)
	chDownload.Wait()
	fmt.Println("\n" + "Synchronization is completed")
}
