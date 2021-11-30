package ssh

import (
	"fmt"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var countGoroutine int

func Sync(conn *ssh.Client, remoteDir string) {
	sc, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatal(err)
	}

	ch := make(chan bool, 50)
	fmt.Println("Synchronization is started")
	listFiles(sc, ch, remoteDir+"/pub/media/", "", 0)

	defer sc.Close()
	defer Disconnect(conn)
}

func listFiles(sc *sftp.Client, ch chan bool, remoteDir, subdir string, isFirst int) (err error) {
	projectPath := paths.GetRunDirPath()
	files, err := sc.ReadDir(remoteDir + subdir)
	if err != nil {
		log.Fatal(err)
	}

	var name string
	for _, f := range files {
		name = f.Name()
		if f.IsDir() {
			if subdir+name != "catalog/product/cache" &&
				subdir+name != "cache" &&
				subdir+name != "images/cache" &&
				subdir+name != "sitemap" &&
				subdir+name != "tmp" &&
				subdir+name != "trashcan" &&
				!strings.Contains(subdir+name+"/", "/cache/") {
				if _, err := os.Stat(projectPath + "/pub/media/" + subdir + name); os.IsNotExist(err) {
					os.Mkdir(projectPath+"/pub/media/"+subdir+name, 0775)
				}
				if countGoroutine <= 3 {
					projectConfig := configs.GetCurrentProjectConfig()
					conn := Connect(projectConfig["SSH_AUTH_TYPE"], projectConfig["SSH_KEY_PATH"], projectConfig["SSH_PASSWORD"], projectConfig["SSH_HOST"], projectConfig["SSH_PORT"], projectConfig["SSH_USERNAME"])
					sc2, err := sftp.NewClient(conn)
					if err != nil {
						fmt.Println(err)
					}
					countGoroutine++
					go listFiles(sc2, ch, remoteDir, subdir+name+"/", isFirst+1)
				} else {
					countGoroutine++
					listFiles(sc, ch, remoteDir, subdir+name+"/", isFirst+1)
				}
			}
		} else if _, err := os.Stat(projectPath + "/pub/media/" + subdir + name); os.IsNotExist(err) {
			ext := strings.ToLower(filepath.Ext(name))
			if ext == ".jpeg" || ext == ".jpg" || ext == ".png" || ext == ".webp" {
				fmt.Printf("%s\n", projectPath+"/pub/media/"+subdir+name)
				downloadFile(sc, remoteDir+"/"+subdir+name, projectPath+"/pub/media/"+subdir+name)
			}
		}
	}

	if isFirst == 0 {
		loop := true
		for loop {
			select {
			case _ = <-ch:
				countGoroutine--
				if 0 == countGoroutine {
					loop = false
				}
			}
		}
		fmt.Println("Synchronization is completed")
	} else if isFirst > 0 {
		ch <- true
	}

	return
}

func downloadFile(sc *sftp.Client, remoteFile, localFile string) (err error) {
	ext := strings.ToLower(filepath.Ext(remoteFile))
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

	isCompressed := false
	switch ext {
	case ".jpg", ".jpeg":
		isCompressed = compressJpg(srcFile, dstFile)
	case ".png":
		isCompressed = compressPng(srcFile, dstFile)
	}

	if !isCompressed {
		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			fmt.Println("Unable to download remote file: " + err.Error() + "\n")
		}
	}

	return
}

func compressJpg(r io.Reader, w io.Writer) bool {
	img, err := jpeg.Decode(r)
	if err != nil {
		return false
	}
	q := jpeg.Options{Quality: 30}
	err = jpeg.Encode(w, img, &q)
	if err != nil {
		return false
	}
	return true
}

func compressPng(r io.Reader, w io.Writer) bool {
	img, err := png.Decode(r)
	if err != nil {
		return false
	}
	enc := png.Encoder{CompressionLevel: -3}
	err = enc.Encode(w, img)
	if err != nil {
		return false
	}
	return true
}
