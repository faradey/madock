package builder

import (
	"github.com/faradey/madock/src/configs/aruntime/nginx"
	"github.com/faradey/madock/src/configs/aruntime/project"
	"github.com/faradey/madock/src/paths"
	"log"
	"os"
	"os/exec"
)

func Up() {
	upNginx()
	upProject()
}

func Down() {
}

func DownAll() {
	downNginx()
}

func upNginx() {
	nginx.MakeConf()
	cmd := exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "up", "--build", "--force-recreate", "--no-deps", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func upProject() {
	projectName := paths.GetRunDirName()
	project.MakeConf(projectName)
	cmd := exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/docker-compose.yml", "up", "--build", "--force-recreate", "--no-deps", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func downNginx() {
	projectName := paths.GetRunDirName()
	cmd := exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/docker-compose.yml", "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	cmd = exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
