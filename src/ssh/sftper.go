package ssh

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"log"
)

func Sync(conn *ssh.Client, remoteDir string) {
	sc, err := sftp.NewClient(conn)
	if err != nil {
		fmt.Println(err)
	}

	listFiles(sc, remoteDir+"/pub/media/")

	defer sc.Close()
	defer Disconnect(conn)
}

func listFiles(sc *sftp.Client, remoteDir string) (err error) {
	files, err := sc.ReadDir(remoteDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		var name, modTime, size string

		name = f.Name()
		modTime = f.ModTime().Format("2006-01-02 15:04:05")
		size = fmt.Sprintf("%12d", f.Size())

		if f.IsDir() {
			name = name + "/"
			modTime = ""
			size = "PRE"
		}
		// Output each file name and size in bytes
		fmt.Println("")
		fmt.Printf("%19s %12s %s\n", modTime, size, name)
	}

	return
}
