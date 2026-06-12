package remote_sync

import (
	"fmt"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"image/jpeg"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

var sc []*sftp.Client

var passwd string

// SSHConfigProvider allows enterprise to customize SSH client configuration.
// For example, to verify host keys via known_hosts or use certificate-based auth.
type SSHConfigProvider interface {
	ClientConfig(host, port, user, password, keyPath string) (*ssh.ClientConfig, error)
}

var sshConfigProvider SSHConfigProvider

// SetSSHConfigProvider sets a custom provider for creating SSH client configurations.
func SetSSHConfigProvider(p SSHConfigProvider) {
	sshConfigProvider = p
}

type RemoteDbStruct struct {
	Host           string `json:"host"`
	Dbname         string `json:"dbname"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	Active         string `json:"active"`
	Model          string `json:"model"`
	Engine         string `json:"engine"`
	InitStatements string `json:"initStatements"`
	Port           string `json:"port"`
}

func ListFiles(chDownload *sync.WaitGroup, ch chan bool, remoteDir, subdir string, indx int, imagesOnly, compress bool) (err error) {
	chDownload.Add(1)
	remainder := indx % len(sc)
	scp := sc[remainder]
	projectConf := configs.GetCurrentProjectConfig()
	projectPath := paths.GetRunDirPath()
	projectMediaPath := projectPath + "/" + projectConf["public_dir"] + "/media/"
	files, err := scp.ReadDir(remoteDir + subdir)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Remote directory not found or not accessible: %q (%s).\nCheck 'ssh/site_root_path' and 'public_dir' in your project config.xml — they must point to the existing media folder on the server.", remoteDir+subdir, err.Error()))
	}

	var name string
	for indx, f := range files {
		name = f.Name()
		subdirName := strings.Trim(subdir+name, "/")
		if f.IsDir() {
			if subdirName != "analytics" &&
				subdirName != "catalog/product/cache" &&
				subdirName != "cache" &&
				subdirName != "captcha" &&
				subdirName != "export" &&
				subdirName != "images/cache" &&
				subdirName != "sitemap" &&
				subdirName != "tmp" &&
				subdirName != "trashcan" &&
				subdirName != "import" &&
				!strings.Contains(subdirName+"/", "/cache") &&
				!strings.Contains(subdirName, ".thumb") {
				if !paths.IsFileExist(projectMediaPath + subdirName) {
					os.Mkdir(projectMediaPath+subdirName, 0775)
				}
				go ListFiles(chDownload, ch, remoteDir, subdirName+"/", indx, imagesOnly, compress)
			}
		} else if !paths.IsFileExist(projectMediaPath + subdirName) {
			ext := strings.ToLower(filepath.Ext(name))
			if !imagesOnly || ext == ".jpeg" || ext == ".jpg" || ext == ".png" || ext == ".webp" {
				remainderDownload := indx % len(sc)
				scpDownload := sc[remainderDownload]
				chDownload.Add(1)
				ch <- true
				go func() {
					DownloadFile(scpDownload, remoteDir+subdirName, projectMediaPath+subdirName, imagesOnly, compress)
					chDownload.Done()
					<-ch
				}()
			}
		}
	}
	chDownload.Done()
	return
}

func DownloadFile(scp *sftp.Client, remoteFile, localFile string, imagesOnly, compress bool) (err error) {
	ext := strings.ToLower(filepath.Ext(remoteFile))
	// Note: SFTP To Go doesn't support O_RDWR mode
	srcFile, err := scp.OpenFile(remoteFile, os.O_RDONLY)
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
	isCompressedOk := compress
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

func Connect(projectConf map[string]string, sshType string) *ssh.Client {

	authType := projectConf[sshType+"/auth_type"]
	if _, ok := projectConf[sshType+"/auth_type"]; !ok {
		authType = projectConf["ssh/auth_type"]
	}

	username := projectConf[sshType+"/username"]
	if _, ok := projectConf[sshType+"/username"]; !ok {
		username = projectConf["ssh/username"]
	}

	port := projectConf[sshType+"/port"]
	if _, ok := projectConf[sshType+"/port"]; !ok {
		port = projectConf["ssh/port"]
	}

	host := projectConf[sshType+"/host"]
	if _, ok := projectConf[sshType+"/host"]; !ok {
		host = projectConf["ssh/host"]
	}

	password := projectConf[sshType+"/password"]
	if _, ok := projectConf[sshType+"/password"]; !ok {
		password = projectConf["ssh/password"]
	}

	keyPath := projectConf[sshType+"/key_path"]
	if _, ok := projectConf[sshType+"/key_path"]; !ok {
		keyPath = projectConf["ssh/key_path"]
	}

	var config *ssh.ClientConfig
	if sshConfigProvider != nil {
		var err error
		config, err = sshConfigProvider.ClientConfig(host, port, username, password, keyPath)
		if err != nil {
			logger.Fatal(err)
		}
	} else {
		config = &ssh.ClientConfig{
			User:            username,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

		if authType == "password" {
			config.Auth = []ssh.AuthMethod{
				ssh.Password(password),
			}
		} else {
			config.Auth = []ssh.AuthMethod{
				publicKey(keyPath),
			}
		}
	}

	conn, err := ssh.Dial("tcp", host+":"+port, config)
	if err != nil {
		logger.Fatal(err)
	}

	return conn
}

func Disconnect(conn *ssh.Client) {
	err := conn.Close()
	if err != nil {
		return
	}
}

func publicKey(path string) ssh.AuthMethod {
	key, err := os.ReadFile(path)
	if err != nil {
		logger.Fatal(err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		if passwd == "" {
			fmt.Print("Input your password for ssh key:")
			var sentence []byte
			sentence, err = terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				logger.Fatalln(err)
			}
			passwd = strings.TrimSpace(string(sentence))
		}
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(passwd))
		if err != nil {
			logger.Fatal(err)
		}
	}

	return ssh.PublicKeys(signer)
}

func RunCommand(conn *ssh.Client, cmd string) string {
	sess, err := conn.NewSession()
	if err != nil {
		logger.Fatal(err)
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		fmt.Println(string(out))
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}

	return string(out)
}

// RunCommandSafe runs a remote command and returns its combined output together
// with any error, without aborting the process. Use it for optional probes
// (e.g. checking whether a binary exists on the remote host).
func RunCommandSafe(conn *ssh.Client, cmd string) (string, error) {
	sess, err := conn.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(cmd)
	return string(out), err
}

func NewClient(conn *ssh.Client) *sftp.Client {
	scTemp, err := sftp.NewClient(conn)
	if err != nil {
		logger.Fatal(err)
	}
	sc = append(sc, scTemp)

	return scTemp
}
