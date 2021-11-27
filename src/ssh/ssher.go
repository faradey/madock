package ssh

import (
	"bytes"
	"fmt"
	"github.com/z7zmey/php-parser/php7"
	"github.com/z7zmey/php-parser/visitor"
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
	sessStdOut := RunCommand(conn, "cat "+remoteDir+"/app/etc/env.php")

	fmt.Println(sessStdOut)

	parser := php7.NewParser([]byte(sessStdOut), "7.4")
	parser.Parse()

	for _, e := range parser.GetErrors() {
		fmt.Println(e)
	}

	var buf bytes.Buffer
	dumper := visitor.Dumper{
		Writer: &buf,
		Indent: "",
	}

	rootNode := parser.GetRootNode()
	dumper.EnterChildNode("db", rootNode)
	rootNode.Walk(&dumper)

	fmt.Println(buf.String())
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
