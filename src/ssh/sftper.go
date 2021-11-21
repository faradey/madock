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

	listFiles(sc, remoteDir+"/pub/media/", "")

	defer sc.Close()
	defer Disconnect(conn)
}

func listFiles(sc *sftp.Client, remoteDir, subdir string) (err error) {
	fmt.Printf("%s\n", remoteDir+subdir)
	files, err := sc.ReadDir(remoteDir + subdir)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		var name string

		name = f.Name()

		if f.IsDir() {
			subdir += "/" + name
			listFiles(sc, remoteDir, subdir)
			fmt.Printf("%s\n", subdir)
		} else {
			fmt.Printf("%s\n", name)
		}
	}

	return
}
