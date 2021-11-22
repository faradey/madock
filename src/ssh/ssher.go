package ssh

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"syscall"
)

var passwd string

func RunCommand(conn *ssh.Client, cmd string) (sessStdOutr, sessStderrr io.Reader) {
	sess, err := conn.NewSession()
	if err != nil {
		panic(err)
	}
	defer sess.Close()
	var sessStdOut io.Writer
	var sessStderr io.Writer
	sessStdOutr, sessStdOut, _ = os.Pipe()
	sessStderrr, sessStderr, _ = os.Pipe()
	sess.Stdout = sessStdOut
	sess.Stderr = sessStderr
	err = sess.Run(cmd)
	if err != nil {
		log.Fatal(err)
	}
	return
}

func DbDump(conn *ssh.Client, remoteDir string) {
	sessStdOut, sessStderr := RunCommand(conn, "cat "+remoteDir+"/app/etc/env.php")
	sessStdOutByte, err := ioutil.ReadAll(sessStdOut)
	if err != nil {
		log.Fatal(err)
	}
	sessStderrByte, err := ioutil.ReadAll(sessStderr)
	if err != nil {
		log.Fatal(err)
	}
	sessStderrText := string(sessStderrByte)
	sessStdOutText := string(sessStdOutByte)
	if len(sessStderrText) > 0 {
		log.Fatal(sessStderrText)
	} else {
		fmt.Println(sessStdOutText)
	}
}

func Connect(keyPath, host, port, username string) *ssh.Client {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			publicKey(keyPath),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", host+":"+port, config)
	if err != nil {
		fmt.Println(err)
	}
	return conn
}

func Disconnect(conn *ssh.Client) {
	conn.Close()
}

func publicKey(path string) ssh.AuthMethod {
	key, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		if passwd == "" {
			fmt.Println("Input your password for ssh key:")
			var sentence []byte
			sentence, err = terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				log.Fatalln(err)
			}
			passwd = strings.TrimSpace(string(sentence))
		}
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(passwd))
		if err != nil {
			log.Fatal(err)
		}
	}
	return ssh.PublicKeys(signer)
}
