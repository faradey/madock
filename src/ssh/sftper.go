package ssh

import (
	"fmt"
	"github.com/faradey/madock/src/configs"
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

	ch := make(chan bool, 50)
	listFiles(sc, ch, remoteDir+"/pub/media/", "", true)

	defer sc.Close()
	defer Disconnect(conn)
}

func listFiles(sc *sftp.Client, ch chan bool, remoteDir, subdir string, isFirst bool) (err error) {
	projectPath := paths.GetRunDirPath()
	dirCount := 0
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
				projectConfig := configs.GetCurrentProjectConfig()
				conn := Connect(projectConfig["SSH_KEY_PATH"], projectConfig["SSH_HOST"], projectConfig["SSH_PORT"], projectConfig["SSH_USERNAME"])
				sc2, err := sftp.NewClient(conn)
				if err != nil {
					fmt.Println(err)
				}
				go listFiles(sc2, ch, remoteDir, subdir+name+"/", false)
				dirCount++
			} else {
				listFiles(sc, ch, remoteDir, subdir+name+"/", false)
			}
		} else {
			if _, err := os.Stat(projectPath + "/pub/media/" + subdir + name); os.IsNotExist(err) {
				fmt.Printf("%s\n", projectPath+"/pub/media/"+subdir+name)
				downloadFile(sc, remoteDir+"/"+subdir+name, projectPath+"/pub/media/"+subdir+name)
			}
		}
	}

	if isFirst == true {
		loop := true
		i := 0
		for loop {
			select {
			case _ = <-ch:
				i++
				if i == dirCount {
					loop = false
				}
			}
		}
		fmt.Println("Synchronization is completed")
	} else {
		ch <- true
	}

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
