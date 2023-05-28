package ssh

import (
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/helper"
	"github.com/faradey/madock/src/paths"
	"github.com/pkg/sftp"
)

var sc []*sftp.Client

func SyncMedia(remoteDir string) {
	var err error
	projectConfig := configs.GetCurrentProjectConfig()
	maxProcs := helper.MaxParallelism() - 1
	var scTemp *sftp.Client
	isFirstConnect := false
	paths.MakeDirsByPath(paths.GetRunDirPath() + "/pub/media")
	for maxProcs > 0 {
		conn := Connect(projectConfig["SSH_AUTH_TYPE"], projectConfig["SSH_KEY_PATH"], projectConfig["SSH_PASSWORD"], projectConfig["SSH_HOST"], projectConfig["SSH_PORT"], projectConfig["SSH_USERNAME"])
		if !isFirstConnect {
			fmt.Println("")
			fmt.Println("Server connection...")
			isFirstConnect = true
		}
		defer Disconnect(conn)
		scTemp, err = sftp.NewClient(conn)
		if err != nil {
			log.Fatal(err)
		}
		defer scTemp.Close()
		sc = append(sc, scTemp)
		maxProcs--
	}

	fmt.Println("\n" + "Synchronization is started")
	ch := make(chan bool, 15)
	var chDownload sync.WaitGroup
	go listFiles(&chDownload, ch, remoteDir+"/pub/media/", "", 0)
	time.Sleep(3 * time.Second)
	chDownload.Wait()
	fmt.Println("\n" + "Synchronization is completed")
}

func SyncFile(remoteDir string) {
	var err error
	path := strings.Trim(attr.Options.Path, "/")
	if path == "" {
		log.Fatal("")
	}
	projectConfig := configs.GetCurrentProjectConfig()
	var sc *sftp.Client
	conn := Connect(projectConfig["SSH_AUTH_TYPE"], projectConfig["SSH_KEY_PATH"], projectConfig["SSH_PASSWORD"], projectConfig["SSH_HOST"], projectConfig["SSH_PORT"], projectConfig["SSH_USERNAME"])
	fmt.Println("")
	fmt.Println("Server connection...")
	defer Disconnect(conn)
	sc, err = sftp.NewClient(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	fmt.Println("\n" + "Synchronization is started")

	downloadFile(sc, strings.TrimRight(remoteDir, "/")+"/"+path, strings.TrimRight(paths.GetRunDirPath(), "/")+"/"+path)
}

func listFiles(chDownload *sync.WaitGroup, ch chan bool, remoteDir, subdir string, indx int) (err error) {
	chDownload.Add(1)
	remainder := indx % len(sc)
	scp := sc[remainder]
	projectPath := paths.GetRunDirPath()
	files, err := scp.ReadDir(remoteDir + subdir)
	if err != nil {
		log.Fatal(err)
	}

	var name string
	for indx, f := range files {
		name = f.Name()
		subdirName := strings.Trim(subdir+name, "/")
		if f.IsDir() {
			if subdirName != "catalog/product/cache" &&
				subdirName != "cache" &&
				subdirName != "images/cache" &&
				subdirName != "sitemap" &&
				subdirName != "tmp" &&
				subdirName != "trashcan" &&
				!strings.Contains(subdirName+"/", "/cache/") &&
				!strings.Contains(subdirName, ".thumb") {
				if _, err := os.Stat(projectPath + "/pub/media/" + subdirName); os.IsNotExist(err) {
					os.Mkdir(projectPath+"/pub/media/"+subdirName, 0775)
				}
				go listFiles(chDownload, ch, remoteDir, subdirName+"/", indx)
			}
		} else if _, err := os.Stat(projectPath + "/pub/media/" + subdirName); os.IsNotExist(err) {
			ext := strings.ToLower(filepath.Ext(name))
			if !attr.Options.ImagesOnly || ext == ".jpeg" || ext == ".jpg" || ext == ".png" || ext == ".webp" {
				remainderDownload := indx % len(sc)
				scpDownload := sc[remainderDownload]
				chDownload.Add(1)
				ch <- true
				go func() {
					downloadFile(scpDownload, remoteDir+subdirName, projectPath+"/pub/media/"+subdirName)
					chDownload.Done()
					<-ch
				}()
			}
		}
	}
	chDownload.Done()
	return
}

func downloadFile(scp *sftp.Client, remoteFile, localFile string) (err error) {
	ext := strings.ToLower(filepath.Ext(remoteFile))
	// Note: SFTP To Go doesn't support O_RDWR mode
	srcFile, err := scp.OpenFile(remoteFile, (os.O_RDONLY))
	if err != nil {
		fmt.Println("\n" + "Unable to open remote file: " + remoteFile + " " + err.Error() + "\n")
		return
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localFile)
	if err != nil {
		fmt.Println("\n" + "Unable to open local file: " + err.Error() + "\n")
		return
	}
	defer dstFile.Close()

	isCompressed := false
	isCompressedOk := attr.Options.Compress
	if isCompressedOk {
		switch ext {
		case ".jpg", ".jpeg":
			isCompressed = compressJpg(srcFile, dstFile)
		}
	}

	if !isCompressed {
		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			fmt.Println("\n" + "Unable to download remote file " + remoteFile + ": " + err.Error() + "\n")
		} else {
			fmt.Printf("\n%s", localFile)
		}
	} else {
		fd, err := dstFile.Stat()
		if err == nil {
			sd, err := srcFile.Stat()
			if err == nil {
				fSize := fd.Size()
				sSize := sd.Size()
				lessOne := (float64(sSize-fSize) / float64(sSize)) * float64(100)
				fmt.Printf("\n%s", localFile)
				fmt.Printf("   (save %d%%)", int(lessOne))
			} else {
				fmt.Println(err)
			}
		} else {
			fmt.Println(err)
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
