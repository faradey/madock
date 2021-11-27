package ssh

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"strings"
	"syscall"
)

var passwd string

func RunCommand(conn *ssh.Client, cmd string) string {
	sess, err := conn.NewSession()
	if err != nil {
		panic(err)
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	return string(out)
}

func DbDump(conn *ssh.Client, remoteDir string) {
	sessStdOut := RunCommand(conn, "php -r \"\\$r1 = include('"+remoteDir+"/app/etc/env.php'); echo json_encode(\\$r1[\"db\"][\"connection\"][\"default\"]);\"")

	fmt.Println(sessStdOut)
}

func Connect(authType, keyPath, pswrd, host, port, username string) *ssh.Client {
	config := &ssh.ClientConfig{
		User:            username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if authType == "password" {
		config.Auth = []ssh.AuthMethod{
			ssh.Password(pswrd),
		}
	} else {
		config.Auth = []ssh.AuthMethod{
			publicKey(keyPath),
		}
	}

	conn, err := ssh.Dial("tcp", host+":"+port, config)
	if err != nil {
		log.Fatal(err)
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
