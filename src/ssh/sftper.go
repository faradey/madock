package ssh

import (
	"fmt"
	"github.com/faradey/madock/src/cli/attr"
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
	"strconv"
	"strings"
)

var countGoroutine int
var sc *sftp.Client
var sc2 *sftp.Client
var sc3 *sftp.Client

func Sync(conn *ssh.Client, remoteDir string) {
	var err error
	sc, err = sftp.NewClient(conn)
	if err != nil {
		log.Fatal(err)
	}

	projectConfig := configs.GetCurrentProjectConfig()
	conn2 := Connect(projectConfig["SSH_AUTH_TYPE"], projectConfig["SSH_KEY_PATH"], projectConfig["SSH_PASSWORD"], projectConfig["SSH_HOST"], projectConfig["SSH_PORT"], projectConfig["SSH_USERNAME"])
	sc2, err = sftp.NewClient(conn2)
	if err != nil {
		fmt.Println(err)
	}
	conn3 := Connect(projectConfig["SSH_AUTH_TYPE"], projectConfig["SSH_KEY_PATH"], projectConfig["SSH_PASSWORD"], projectConfig["SSH_HOST"], projectConfig["SSH_PORT"], projectConfig["SSH_USERNAME"])
	sc3, err = sftp.NewClient(conn3)
	if err != nil {
		fmt.Println(err)
	}
	countGoroutine = 0
	ch := make(chan bool, 50)
	fmt.Println("")
	fmt.Println("Synchronization is started")
	listFiles(ch, remoteDir+"/pub/media/", "", 0)

	defer sc.Close()
	defer sc2.Close()
	defer sc3.Close()
	defer Disconnect(conn)
	defer Disconnect(conn2)
	defer Disconnect(conn3)
}

func listFiles(ch chan bool, remoteDir, subdir string, isFirst int) (err error) {
	scp := sc
	remainder := countGoroutine % 3
	if remainder == 1 {
		scp = sc2
	} else if remainder == 2 {
		scp = sc3
	}
	projectPath := paths.GetRunDirPath()
	files, err := scp.ReadDir(remoteDir + subdir)
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

				if countGoroutine <= 5 || isFirst == 0 {
					countGoroutine++
					go listFiles(ch, remoteDir, subdir+name+"/", isFirst+1)
				} else {
					countGoroutine++
					listFiles(ch, remoteDir, subdir+name+"/", isFirst+1)
				}
			}
		} else if _, err := os.Stat(projectPath + "/pub/media/" + subdir + name); os.IsNotExist(err) {
			ext := strings.ToLower(filepath.Ext(name))
			isImagesOnly := attr.Attributes["--images-only"]
			if isImagesOnly == "" || ext == ".jpeg" || ext == ".jpg" || ext == ".png" || ext == ".webp" {
				fmt.Printf("\n%s", projectPath+"/pub/media/"+subdir+name)
				downloadFile(scp, remoteDir+"/"+subdir+name, projectPath+"/pub/media/"+subdir+name)
			}
		}
	}

	if isFirst == 0 {
		loop := true
		for loop {
			select {
			case _ = <-ch:
				countGoroutine--
				if 0 >= countGoroutine {
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

func downloadFile(scp *sftp.Client, remoteFile, localFile string) (err error) {
	ext := strings.ToLower(filepath.Ext(remoteFile))
	// Note: SFTP To Go doesn't support O_RDWR mode
	srcFile, err := scp.OpenFile(remoteFile, (os.O_RDONLY))
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
	isCompressedOk := attr.Attributes["--compress"]
	if isCompressedOk != "" {
		switch ext {
		case ".jpg", ".jpeg":
			isCompressed = compressJpg(srcFile, dstFile)
		case ".png":
			isCompressed = compressPng(srcFile, dstFile)
		}
	}

	if !isCompressed {
		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			fmt.Println("Unable to download remote file: " + err.Error() + "\n")
		}
	} else {
		fd, err := dstFile.Stat()
		if err == nil {
			sd, err := srcFile.Stat()
			if err == nil {
				fSize := fd.Size()
				sSize := sd.Size()
				fmt.Printf("  (saved %v)", strconv.FormatFloat(float64((sSize-fSize)/(sSize*1.0)*(100.0)), 'f', -1, 32))
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
