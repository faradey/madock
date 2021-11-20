package ssh

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"os"
)

var conn *ssh.Client

func RunCommand(cmd string) {
	sess, err := conn.NewSession()
	if err != nil {
		panic(err)
	}
	defer sess.Close()
	sessStdOut, err := sess.StdoutPipe()
	if err != nil {
		panic(err)
	}
	go io.Copy(os.Stdout, sessStdOut)
	sessStderr, err := sess.StderrPipe()
	if err != nil {
		panic(err)
	}
	go io.Copy(os.Stderr, sessStderr)
	err = sess.Run(cmd) // eg., /usr/bin/whoami
	if err != nil {
		panic(err)
	}
}

func Connect(keyPath string) {
	config := &ssh.ClientConfig{
		User: "username",
		Auth: []ssh.AuthMethod{
			publicKey(keyPath),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	var err error
	conn, err = ssh.Dial("tcp", "host", config)
	if err != nil {
		fmt.Println(err)
	}
}

func Disconnect() {
	conn.Close()
}

func publicKey(path string) ssh.AuthMethod {
	key, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatal(err)
	}
	return ssh.PublicKeys(signer)
}
