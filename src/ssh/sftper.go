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

	var name string
	for _, f := range files {
		name = f.Name()

		if f.IsDir() {
			listFiles(sc, remoteDir, subdir+name+"/")
			fmt.Printf("%s\n", subdir)
		} else {
			fmt.Printf("%s\n", name)
		}
	}

	return
}
