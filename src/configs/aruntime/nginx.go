package aruntime

import (
	"github.com/faradey/madock/src/paths"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strconv"
	"strings"
)

func CreateDefaultNginxConf() {
	/* Create nginx default configuration for Magento2 */
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/conf/default-proxy.conf"
	allFileData := ""
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}
	str := string(b)
	projectsNames := paths.GetDirs(paths.GetExecDirPath() + "/aruntime/projects")
	for index, name := range projectsNames {
		if _, err := os.Stat(paths.GetExecDirPath() + "/aruntime/projects/" + name + "/stopped"); os.IsNotExist(err) {
			strReplaced := strings.Replace(str, "{{{NGINX_PORT}}}", strconv.Itoa(index+17000), -1)
			strReplaced = strings.Replace(strReplaced, "{{{HOST_NAMES}}}", "loc."+name+".com", -1)
			allFileData += strReplaced
		}
	}

	nginxFile := paths.GetExecDirPath() + "/aruntime/ctx/proxy.conf"
	err = ioutil.WriteFile(nginxFile, []byte(allFileData), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx default configuration for Magento2 */
}

func CreateDefaultNginxDockerfile() {
	/* Create nginx Dockerfile configuration */
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/proxy.Dockerfile"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}

	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
	err = ioutil.WriteFile(paths.GetExecDirPath()+"/aruntime/ctx/Dockerfile", b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx Dockerfile configuration */
}

func CreateDefaultNginxDockerCompose() {
	/* Copy nginx docker-compose configuration */
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/docker-compose-proxy.yml"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(paths.GetExecDirPath()+"/aruntime/docker-compose.yml", b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx Dockerfile configuration */
}

func CreateProjectNginxConf(projectName string, generalConf map[string]string) {
	/* Create nginx default configuration for Magento2 */
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/conf/default.conf"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}
	str := string(b)
	str = strings.Replace(str, "{{{NGINX_PORT}}}", generalConf["NGINX_PORT"], -1)
	str = strings.Replace(str, "{{{HOST_NAMES}}}", "loc."+projectName+".com", -1)
	str = strings.Replace(str, "{{{PROJECT_NAME}}}", projectName, -1)
	str = strings.Replace(str, "{{{HOST_NAMES_WEBSITES}}}", "loc."+projectName+".com base;", -1)
	nginxFile := paths.GetExecDirPath() + "/projects/" + projectName + "/docker/nginx/" + projectName + ".conf"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx default configuration for Magento2 */
}

func CreateProjectNginxDockerfile() {
	/* Create nginx Dockerfile configuration */
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/Dockerfile"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}

	projectsNames := paths.GetDirs(paths.GetExecDirPath() + "/aruntime/projects")
	copyLines := ""
	for _, name := range projectsNames {
		copyLines += "COPY ./" + name + ".conf /etc/nginx/sites-enabled/" + name + ".conf\n"
	}

	str := string(b)
	str = strings.Replace(str, "{{{COPY_NGINX_CONF}}}", copyLines, -1)
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	str = strings.Replace(str, "{{{UID}}}", usr.Uid, -1)
	str = strings.Replace(str, "{{{GUID}}}", usr.Gid, -1)
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime")
	err = ioutil.WriteFile(paths.GetExecDirPath()+"/aruntime/ctx/Dockerfile", []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx Dockerfile configuration */
}

func CreateProjectNginxDockerCompose() {
	/* Copy nginx docker-compose configuration */
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/docker-compose.yml"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)

	projectsDirs := paths.GetDirs(paths.GetExecDirPath() + "/aruntime/projects")

	volumes := ""

	for _, dir := range projectsDirs {
		volumes += "      - ./projects/" + dir + "/src/:/var/www/html/" + dir + "/\n"
		nginxConfFile := paths.GetExecDirPath() + "/projects/" + dir + "/docker/nginx/" + dir + ".conf"
		if _, err := os.Stat(nginxConfFile); os.IsNotExist(err) {
			log.Fatal(err)
		}
		confFileData, err := os.ReadFile(nginxConfFile)
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile(paths.GetExecDirPath()+"/aruntime/ctx/"+dir+".conf", confFileData, 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}
	}

	str = strings.Replace(str, "{{{VOLUMES}}}", volumes, -1)
	err = ioutil.WriteFile(paths.GetExecDirPath()+"/aruntime/docker-compose.yml", []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx Dockerfile configuration */
}
