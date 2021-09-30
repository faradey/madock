package nginx

import (
	"github.com/faradey/madock/src/paths"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

func MakeConf() {
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + paths.GetRunDirName() + "/docker/nginx")
	makeProxy()
	makeDockerfile()
	makeDockerCompose()
}

func makeProxy() {
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

func makeDockerfile() {
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

func makeDockerCompose() {
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
