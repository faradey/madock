package builder

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/configs/aruntime/nginx"
	"github.com/faradey/madock/src/configs/aruntime/project"
	"github.com/faradey/madock/src/paths"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func Up() {
	upNginx()
	upProject()
}

func UpWithBuild() {
	upNginx()
	upProjectWithBuild()
}

func Down() {
	projectName := paths.GetRunDirName()
	cmd := exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/docker-compose.yml", "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func DownAll() {
	Down()
	downNginx()
}

func upNginx() {
	nginx.MakeConf()
	cmd := exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "up" /*, "--build"*/, "--force-recreate", "--no-deps", "-d")
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
	cmd := exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/docker-compose.yml", "up", "--no-deps", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	projectConfig := configs.GetProjectConfig()
	if val, ok := projectConfig["CRON_ENABLED"]; ok && val == "true" {
		Cron("--on")
	} else {
		Cron("--off")
	}
}

func upProjectWithBuild() {
	projectName := paths.GetRunDirName()
	project.MakeConf(projectName)
	cmd := exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/docker-compose.yml", "up", "--build", "--force-recreate", "--no-deps", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	projectConfig := configs.GetProjectConfig()
	if val, ok := projectConfig["CRON_ENABLED"]; ok && val == "true" {
		Cron("--on")
	} else {
		Cron("--off")
	}
}

func downNginx() {
	cmd := exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Magento(flag string) {
	projectName := paths.GetRunDirName()
	cmd := exec.Command("docker", "exec", "-i", "-u", "www-data", projectName+"-php-1", "bash", "-c", "cd /var/www/html && php bin/magento "+flag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Composer(flag string) {
	projectName := paths.GetRunDirName()
	cmd := exec.Command("docker", "exec", "-i", "-u", "www-data", projectName+"-php-1", "bash", "-c", "cd /var/www/html && composer "+flag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func DbImport(option string) {
	if len(option) > 0 && option != "-f" {
		option = ""
	}
	projectName := paths.GetRunDirName()
	projectConfig := configs.GetProjectConfig()
	dbsPath := paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db"
	dbNames := paths.GetFiles(dbsPath)
	for index, dbName := range dbNames {
		fmt.Println(strconv.Itoa(index+1) + ") " + dbName)
	}
	fmt.Println("Choose one of the options offered")
	buf := bufio.NewReader(os.Stdin)
	sentence, err := buf.ReadBytes('\n')
	selected := strings.TrimSpace(string(sentence))
	selectedInt := 0
	if err != nil {
		log.Fatalln(err)
	} else {
		selectedInt, err = strconv.Atoi(selected)
		if err != nil {
			log.Fatal(err)
		}

		if selectedInt > len(dbNames) {
			log.Fatal("The item you selected was not found")
		}
	}

	ext := dbNames[selectedInt-1][len(dbNames[selectedInt-1])-2:]
	out := &gzip.Reader{}

	selectedFile, err := os.Open(dbsPath + "/" + dbNames[selectedInt-1])
	if err != nil {
		log.Fatal(err)
	}
	defer selectedFile.Close()

	var cmd *exec.Cmd
	if option != "" {
		cmd = exec.Command("docker", "exec", "-i", "-u", "mysql", projectName+"-db-1", "mysql", option, "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-h", "db", projectConfig["DB_DATABASE"])
	} else {
		cmd = exec.Command("docker", "exec", "-i", "-u", "mysql", projectName+"-db-1", "mysql", "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-h", "db", projectConfig["DB_DATABASE"])
	}

	if ext == "gz" {
		out, err = gzip.NewReader(selectedFile)
		if err != nil {
			log.Fatal(err)
		}
		cmd.Stdin = out
	} else {
		cmd.Stdin = selectedFile
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database import completed successfully")
}

func DbExport() {
	projectName := paths.GetRunDirName()
	projectConfig := configs.GetProjectConfig()
	dbsPath := paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db/"
	selectedFile, err := os.Create(dbsPath + time.Now().Format("2006-01-02_15-04-05") + ".sql.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer selectedFile.Close()
	writer := gzip.NewWriter(selectedFile)
	cmd := exec.Command("docker", "exec", "-i", "-u", "mysql", projectName+"-db-1", "mysqldump", "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-h", "db", projectConfig["DB_DATABASE"])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdout = writer
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database export completed successfully")
}

func Cron(flag string) {
	projectName := paths.GetRunDirName()
	var cmd *exec.Cmd
	if flag == "--on" {
		cmd = exec.Command("docker", "exec", "-i", "-u", "root", projectName+"-php-1", "service", "cron", "start")
		cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", projectName+"-php-1", "bash", "-c", "cd /var/www/html && php bin/magento cron:install &&  php bin/magento cron:run")
		cmdSub.Stdout = os.Stdout
		cmdSub.Stderr = os.Stderr
		err := cmdSub.Run()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cmd = exec.Command("docker", "exec", "-i", "-u", "root", projectName+"-php-1", "service", "cron", "stop")
		cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", projectName+"-php-1", "bash", "-c", "cd /var/www/html && php bin/magento cron:remove")
		cmdSub.Stdout = os.Stdout
		cmdSub.Stderr = os.Stderr
		err := cmdSub.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Bash(containerName string, isRoot bool) {
	projectName := paths.GetRunDirName()
	var cmd *exec.Cmd
	/*if isRoot {
		cmd = exec.Command("docker", "exec", "-it", "-u", "root", projectName+"-"+containerName+"-1", "bash")
	} else {*/
	cmd = exec.Command("docker", "exec", "-it", projectName+"-"+containerName+"-1", "bash")
	//}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
