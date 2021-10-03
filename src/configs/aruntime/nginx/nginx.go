package nginx

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

func MakeConf() {
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + paths.GetRunDirName() + "/docker/nginx")
	setPorts()
	makeProxy()
	makeDockerfile()
	makeDockerCompose()
}

func setPorts() {
	projects := paths.GetDirs(paths.GetExecDirPath() + "/aruntime/projects")
	portsFile := paths.GetExecDirPath() + "/aruntime/ports.conf"
	portsConfig := make(map[string]string)
	if _, err := os.Stat(portsFile); os.IsNotExist(err) {
		lines := ""
		for port, line := range projects {
			lines += line + "=" + strconv.Itoa(17000+port) + "\n"
		}
		err = ioutil.WriteFile(portsFile, []byte(lines), 0755)
	}

	portsConfig = configs.ParseFile(portsFile)

	if _, ok := portsConfig[paths.GetRunDirName()]; !ok {
		f, err := os.OpenFile(portsFile,
			os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Println(err)
		}
		defer f.Close()
		maxPort := getMaxPort(portsConfig)
		if _, err := f.WriteString(paths.GetRunDirName() + "=" + strconv.Itoa(maxPort+1) + "\n"); err != nil {
			log.Println(err)
		}
	}
}

func makeProxy() {
	portsFile := paths.GetExecDirPath() + "/aruntime/ports.conf"
	portsConfig := configs.ParseFile(portsFile)
	/* Create nginx default configuration for Magento2 */
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/conf/default-proxy.conf"
	allFileData := ""
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}
	str := string(b)
	projectsNames := paths.GetDirs(paths.GetExecDirPath() + "/aruntime/projects")
	for _, name := range projectsNames {
		if _, err := os.Stat(paths.GetExecDirPath() + "/aruntime/projects/" + name + "/stopped"); os.IsNotExist(err) {
			strReplaced := strings.Replace(str, "{{{NGINX_PORT}}}", portsConfig[name], -1)
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

func getMaxPort(conf map[string]string) int {
	max := 0
	portInt := 0
	var err error
	for _, port := range conf {
		portInt, err = strconv.Atoi(port)
		if err != nil {
			log.Fatal(err)
		}
		if max < portInt {
			max = portInt
		}
	}

	return max
}
