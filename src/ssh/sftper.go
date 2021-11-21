package ssh

import (
	"fmt"
	"github.com/faradey/madock/src/paths"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
)

func Sync(conn *ssh.Client, remoteDir string) {
	sc, err := sftp.NewClient(conn)
	if err != nil {
		fmt.Println(err)
	}

	listFiles(sc, remoteDir+"/pub/media/", "", true)

	defer sc.Close()
	defer Disconnect(conn)
}

func listFiles(sc *sftp.Client, remoteDir, subdir string, isFirst bool) (err error) {
	projectPath := paths.GetRunDirPath()
	files, err := sc.ReadDir(remoteDir + subdir)
	if err != nil {
		log.Fatal(err)
	}

	var name string
	for _, f := range files {
		name = f.Name()
		if f.IsDir() {
			if _, err := os.Stat(projectPath + "/pub/media/" + subdir + name); os.IsNotExist(err) {
				os.Mkdir(projectPath+"/pub/media/"+subdir+name, 0775)
			}
			if isFirst == true {
				go listFiles(sc, remoteDir, subdir+name+"/", false)
			} else {
				listFiles(sc, remoteDir, subdir+name+"/", false)
			}
		} else {
			if _, err := os.Stat(projectPath + "/pub/media/" + subdir + name); os.IsNotExist(err) {
				downloadFile(sc, remoteDir+"/"+subdir+name, projectPath+"/pub/media/"+subdir+name)
			}
		}
	}

	fmt.Println("Synchronization will run in the background")

	return
}

func downloadFile(sc *sftp.Client, remoteFile, localFile string) (err error) {
	// Note: SFTP To Go doesn't support O_RDWR mode
	srcFile, err := sc.OpenFile(remoteFile, (os.O_RDONLY))
	if err != nil {
		fmt.Println("Unable to open remote file: " + err.Error() + "\n")
		return
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localFile)
	if err != nil {
		fmt.Println("Unable to open local file: " + err.Error() + "\n")
		return
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		fmt.Println("Unable to download remote file: " + err.Error() + "\n")
	}

	return
}
