package builder

import (
	"fmt"
	"github.com/faradey/madock/src/paths"
	"log"
	"os/exec"
)

func Build() {
	UpNginx()
	DownNginx()
}

func UpNginx() {
	cmd := exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "up", "--build", "--force-recreate", "--no-deps", "-d")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(output))
}

func DownNginx() {
	cmd := exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "down")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(output))
}
