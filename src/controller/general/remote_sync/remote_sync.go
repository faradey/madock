package remote_sync

import (
	"errors"
	"fmt"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
	"image/jpeg"
	"io"
	"log"
	"net"
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

		config.Auth = authMethods(authType, password, keyPath)
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

// authMethods builds what the handshake is allowed to try.
//
// Public-key authentication is **one** method, not two. The agent and the key
// file were offered as separate entries in config.Auth, and to the protocol both
// answer to the name "publickey" — which is what the client marks as tried, by
// name. So the agent refusing closed the path to ssh/key_path as well: the
// configured key was never offered at all. Measured on a live sshd, the server
// log held exactly one attempt, with a key from the agent that it did not
// accept, and none with the key the project had been told to use.
//
// The visible symptom was `ssh host` working while `remote:sync:*` failed on the
// same host, which is precisely the difference this code exists to avoid.
func authMethods(authType, password, keyPath string) []ssh.AuthMethod {
	if authType == "password" {
		return []ssh.AuthMethod{ssh.Password(password)}
	}

	return []ssh.AuthMethod{publicKeyAuth(keyPath)}
}

// publicKeyAuth offers the agent's keys and the configured key file as a single
// publickey method, agent first, so every key is tried.
//
// The callback runs at handshake time, so a key added to the agent after the
// config was built still counts. The key file is not read while an agent key is
// still being tried: the signer built for it knows its own public half, and the
// private half — and the passphrase prompt with it — waits until the server has
// said it would accept that key.
func publicKeyAuth(keyPath string) ssh.AuthMethod {
	return ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
		var signers []ssh.Signer

		if conn := dialAgent(); conn != nil {
			agentSigners, err := agent.NewClient(conn).Signers()
			if err == nil {
				signers = append(signers, agentSigners...)
			}
			// An agent that cannot be talked to is no reason to give up the key
			// file, which is the whole point of putting them in one method.
		}

		if keyPath != "" {
			if signer := fileSigner(keyPath); signer != nil {
				signers = append(signers, signer)
			}
		}

		if len(signers) == 0 {
			// Said here rather than left to the handshake, which would report
			// only that no method remained.
			if keyPath == "" {
				return nil, fmt.Errorf("no ssh keys to offer: the agent has none and no key_path is configured")
			}
			return nil, fmt.Errorf("no ssh keys to offer: the agent has none and the public half of %s could not be read", keyPath)
		}

		return signers, nil
	})
}

// dialAgent connects to the running SSH agent, or reports none. A socket left
// behind by a dead agent counts as none — the key file is the better answer
// there than an error.
func dialAgent() net.Conn {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil
	}

	return conn
}

// fileSigner wraps a key file as a signer whose public half is known up front
// and whose private half is read on the first signature asked of it.
//
// nil when the public half cannot be established without a passphrase. Two
// reasons, and the second is not a style preference: an unreadable fallback key
// must not sink a connection the agent may still complete, and a signer with no
// public key is dereferenced by the handshake before anything can check it —
// nil there is a panic, not a skip.
func fileSigner(path string) ssh.Signer {
	public := publicKeyOf(path)
	if public == nil {
		return nil
	}

	return &lazyFileSigner{path: path, public: public}
}

// publicKeyOf finds the public half of a key file without ever asking for a
// passphrase: from the .pub beside it, from the key itself when it is not
// encrypted, or from the cleartext public key an encrypted OpenSSH key carries
// alongside its encrypted private half.
func publicKeyOf(path string) ssh.PublicKey {
	if pub, err := os.ReadFile(path + ".pub"); err == nil {
		if key, _, _, _, err := ssh.ParseAuthorizedKey(pub); err == nil {
			return key
		}
	}

	key, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err == nil {
		return signer.PublicKey()
	}

	var missing *ssh.PassphraseMissingError
	if errors.As(err, &missing) {
		return missing.PublicKey
	}

	return nil
}

// lazyFileSigner signs with a key file, reading and decrypting it on first use.
// Sign is reached only after the server has accepted the public key, so a
// passphrase is asked for where it is needed rather than while an agent key is
// still in play.
type lazyFileSigner struct {
	path   string
	public ssh.PublicKey

	once   sync.Once
	signer ssh.Signer
	err    error
}

func (l *lazyFileSigner) PublicKey() ssh.PublicKey {
	return l.public
}

func (l *lazyFileSigner) Sign(rand io.Reader, data []byte) (*ssh.Signature, error) {
	signer, err := l.load()
	if err != nil {
		return nil, err
	}

	return signer.Sign(rand, data)
}

// SignWithAlgorithm is what makes an RSA key file usable at all. Without it the
// handshake treats this as a plain Signer, which it assumes can only produce
// ssh-rsa — SHA-1, refused by every OpenSSH since 8.8 — and the key is rejected
// for a reason that has nothing to do with the key.
func (l *lazyFileSigner) SignWithAlgorithm(rand io.Reader, data []byte, algorithm string) (*ssh.Signature, error) {
	signer, err := l.load()
	if err != nil {
		return nil, err
	}

	as, ok := signer.(ssh.AlgorithmSigner)
	if !ok {
		if algorithm != "" && algorithm != signer.PublicKey().Type() {
			return nil, fmt.Errorf("ssh: key %s cannot sign with %s", l.path, algorithm)
		}
		return signer.Sign(rand, data)
	}

	return as.SignWithAlgorithm(rand, data, algorithm)
}

func (l *lazyFileSigner) load() (ssh.Signer, error) {
	l.once.Do(func() {
		l.signer, l.err = loadSigner(l.path)
	})

	return l.signer, l.err
}

// loadSigner reads and parses a key file, asking for the passphrase when it is
// encrypted. One place prompts, so the answer is remembered once.
func loadSigner(path string) (ssh.Signer, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err == nil {
		return signer, nil
	}

	var missing *ssh.PassphraseMissingError
	if !errors.As(err, &missing) {
		return nil, err
	}

	if passwd == "" {
		fmt.Print("Input your password for ssh key:")
		sentence, readErr := terminal.ReadPassword(int(syscall.Stdin))
		if readErr != nil {
			return nil, readErr
		}
		passwd = strings.TrimSpace(string(sentence))
	}

	return ssh.ParsePrivateKeyWithPassphrase(key, []byte(passwd))
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
