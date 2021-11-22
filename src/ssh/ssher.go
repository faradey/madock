package ssh

import (
	"bytes"
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

func RunCommand(conn *ssh.Client, cmd string) (sessStdOutText, sessStderrText string) {
	sess, err := conn.NewSession()
	if err != nil {
		panic(err)
	}
	defer sess.Close()
	sessStdOutr, sessStdOut, _ := os.Pipe()
	sessStderrr, sessStderr, _ := os.Pipe()
	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr
	err = sess.Run(cmd)
	if err != nil {
		log.Fatal(err)
	}
	var sessStdOutByte bytes.Buffer
	io.Copy(&sessStdOutByte, sessStdOutr)

	var sessStderrByte bytes.Buffer
	io.Copy(&sessStdOutByte, sessStderrr)

	sessStderrText = sessStderrByte.String()
	sessStdOutText = sessStdOutByte.String()
	sessStdOut.Close()
	sessStderr.Close()
	return
}

func DbDump(conn *ssh.Client, remoteDir string) {
	sessStdOut, sessStderr := RunCommand(conn, "cat "+remoteDir+"/app/etc/env.php")

	if len(sessStderr) > 0 {
		log.Fatal(sessStderr)
	} else {
		fmt.Println(sessStdOut)
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
