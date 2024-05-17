package file

import (
	"fmt"
	"github.com/faradey/madock/src/controller/general/remote_sync"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/pkg/sftp"
	"strings"
)

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralRemoteSyncFile)).(*arg_struct.ControllerGeneralRemoteSyncFile)

	projectConf := configs.GetCurrentProjectConfig()
	var err error
	path := strings.Trim(args.Path, "/")
	if path == "" {
		logger.Fatal("Path is empty")
	}
	var sc *sftp.Client
	sshType := "ssh"
	if args.SshType != "" {
		sshType += "_" + args.SshType
	}
	siteRootPath := projectConf[sshType+"/site_root_path"]
	if _, ok := projectConf[sshType+"/site_root_path"]; !ok {
		siteRootPath = projectConf["ssh/site_root_path"]
	}
	conn := remote_sync.Connect(projectConf, sshType)
	fmt.Println("")
	fmt.Println("Server connection...")
	defer remote_sync.Disconnect(conn)
	sc, err = sftp.NewClient(conn)
	if err != nil {
		logger.Fatal(err)
	}
	defer sc.Close()

	fmt.Println("\n" + "Synchronization is started")

	remote_sync.DownloadFile(sc, strings.TrimRight(siteRootPath, "/")+"/"+path, strings.TrimRight(paths.GetRunDirPath(), "/")+"/"+path, false, false)
}
