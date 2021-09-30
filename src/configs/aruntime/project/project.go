package project

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strings"
)

func MakeConf(projectName string) {
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects/" + projectName)
	src := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/src"
	if _, err := os.Lstat(src); err == nil {
		if err := os.Remove(src); err != nil {
			log.Fatalf("failed to unlink: %+v", err)
		}
	}

	err := os.Symlink(paths.GetRunDirPath(), src)
	if err != nil {
		log.Fatal(err)
	}

	makeNginxDockerfile(projectName)
	makeNginxConf(projectName)
	makePhpDockerfile(projectName)
}

func makeNginxDockerfile(projectName string) {
	dockerDefFile := paths.GetExecDirPath() + "/docker/nginx/Dockerfile"
	if _, err := os.Stat(dockerDefFile); os.IsNotExist(err) {
		log.Fatal(err)
	}

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx")
	nginxFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/nginx.Dockerfile"
	err = ioutil.WriteFile(nginxFile, b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeNginxConf(projectName string) {
	defFile := paths.GetExecDirPath() + "/projects/" + projectName + "/docker/nginx/default.conf"
	if _, err := os.Stat(defFile); os.IsNotExist(err) {
		log.Fatal(err)
	}

	b, err := os.ReadFile(defFile)
	if err != nil {
		log.Fatal(err)
	}

	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx")
	nginxFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/nginx.conf"
	err = ioutil.WriteFile(nginxFile, b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makePhpDockerfile(projectName string) {
	dockerDefFile := paths.GetExecDirPath() + "/docker/php/Dockerfile"
	if _, err := os.Stat(dockerDefFile); os.IsNotExist(err) {
		log.Fatal(err)
	}

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	projectConf := configs.GetProjectConfig()
	str := string(b)
	str = strings.Replace(str, "{{{PHP_VERSION}}}", projectConf["PHP_VERSION"], -1)
	str = strings.Replace(str, "{{{PHP_TZ}}}", projectConf["PHP_TZ"], -1)
	str = strings.Replace(str, "{{{PHP_MODULE_XDEBUG}}}", projectConf["PHP_MODULE_XDEBUG"], -1)
	str = strings.Replace(str, "{{{PHP_XDEBUG_REMOTE_HOST}}}", projectConf["PHP_XDEBUG_REMOTE_HOST"], -1)
	str = strings.Replace(str, "{{{PHP_MODULE_IONCUBE}}}", projectConf["PHP_MODULE_IONCUBE"], -1)
	usr, err := user.Current()
	if err == nil {
		str = strings.Replace(str, "{{{UID}}}", usr.Uid, -1)
		str = strings.Replace(str, "{{{GUID}}}", usr.Gid, -1)
	} else {
		log.Fatal(err)
	}
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx")
	nginxFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/php.Dockerfile"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}
