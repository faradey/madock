package project

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
)

func MakeConf(projectName string) {
	src := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName) + "/src"
	if _, err := os.Lstat(src); err == nil {
		if err := os.Remove(src); err != nil {
			log.Fatalf("failed to unlink: %+v", err)
		}
	}

	err := os.Symlink(paths.GetRunDirPath(), src)
	if err != nil {
		log.Fatal(err)
	}

	makeDockerCompose(projectName)
	makeNginxDockerfile(projectName)
	makeNginxConf(projectName)
	makePhpDockerfile(projectName)
	makeDBDockerfile(projectName)
	makeElasticDockerfile(projectName)
	makeRedisDockerfile(projectName)
	makeNodeDockerfile(projectName)
	makeKibanaConf(projectName)
	makeScriptsConf(projectName)
}

func makeScriptsConf(projectName string) {
	exPath := paths.GetExecDirPath()
	src := exPath + "/aruntime/projects/" + projectName + "/ctx/scripts"
	if fi, err := os.Lstat(src); err == nil {
		if fi.Mode()&os.ModeSymlink != os.ModeSymlink {
			err := os.RemoveAll(src)
			if err == nil {
				err := os.Symlink(exPath+"/scripts", src)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				fmt.Println(err)
			}
		}
	} else {
		err := os.Symlink(exPath+"/scripts", src)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func makeKibanaConf(projectName string) {
	file := getDockerConfigFile(projectName, "/docker/kibana/kibana.yml")

	b, err := os.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	str := string(b)
	str = configs.ReplaceConfigValue(str)

	filePath := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/kibana.yml"
	err = ioutil.WriteFile(filePath, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeNginxDockerfile(projectName string) {
	dockerDefFile := getDockerConfigFile(projectName, "/docker/nginx/Dockerfile")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}
	str := string(b)
	str = configs.ReplaceConfigValue(str)

	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/nginx.Dockerfile"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeNginxConf(projectName string) {
	defFile := getDockerConfigFile(projectName, "/docker/nginx/conf/default.conf")

	b, err := os.ReadFile(defFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	projectConf := configs.GetCurrentProjectConfig()
	str = configs.ReplaceConfigValue(str)
	hostName := "loc." + projectName + ".com"
	hostNameWebsites := "loc." + projectName + ".com base;"
	if val, ok := projectConf["HOSTS"]; ok {
		var onlyHosts []string
		var websitesHosts []string
		hosts := strings.Split(val, " ")
		if len(hosts) > 0 {
			for _, hostAndStore := range hosts {
				onlyHosts = append(onlyHosts, strings.Split(hostAndStore, ":")[0])
				if len(strings.Split(hostAndStore, ":")) > 1 {
					websitesHosts = append(websitesHosts, strings.Split(hostAndStore, ":")[0]+" "+strings.Split(hostAndStore, ":")[1]+";")
				}
			}
			if len(onlyHosts) > 0 {
				hostName = strings.Join(onlyHosts, "\n")
			}
			if len(websitesHosts) > 0 {
				hostNameWebsites = strings.Join(websitesHosts, "\n")
			}
		}
	}
	str = strings.Replace(str, "{{{HOST_NAMES}}}", hostName, -1)
	str = strings.Replace(str, "{{{PROJECT_NAME}}}", projectName, -1)
	str = strings.Replace(str, "{{{HOST_NAMES_WEBSITES}}}", hostNameWebsites, -1)

	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx")
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/nginx.conf"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makePhpDockerfile(projectName string) {
	dockerDefFile := getDockerConfigFile(projectName, "/docker/php/Dockerfile")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs.ReplaceConfigValue(str)
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/php.Dockerfile"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	dockerDefFile = getDockerConfigFile(projectName, "/docker/php/DockerfileWithoutXdebug")

	b, err = os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str = string(b)
	str = configs.ReplaceConfigValue(str)
	nginxFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/php.DockerfileWithoutXdebug"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeDockerCompose(projectName string) {
	overrideFile := runtime.GOOS
	projectConf := configs.GetCurrentProjectConfig()

	dockerDefFile := getDockerConfigFile(projectName, "/docker/docker-compose.yml")
	dockerDefFileForOS := getDockerConfigFile(projectName, "/docker/docker-compose."+overrideFile+".yml")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	portsConfig := configs.ParseFile(paths.GetExecDirPath() + "/aruntime/ports.conf")
	portNumber, err := strconv.Atoi(portsConfig[projectName])
	if err != nil {
		log.Fatal(err)
	}

	portNumberRanged := (portNumber - 1) * 20
	hostName := "loc." + projectName + ".com"
	if val, ok := projectConf["HOSTS"]; ok {
		hosts := strings.Split(val, " ")
		if len(hosts) > 0 {
			hostName = strings.Split(hosts[0], ":")[0]
		}
	}
	str = configs.ReplaceConfigValue(str)
	str = strings.Replace(str, "{{{HOST_NAME_DEFAULT}}}", hostName, -1)
	str = strings.Replace(str, "{{{NGINX_PROJECT_PORT}}}", strconv.Itoa(portNumberRanged+17000), -1)
	str = strings.Replace(str, "{{{NGINX_PROJECT_PORT_SSL}}}", strconv.Itoa(portNumberRanged+17001), -1)
	for i := 2; i < 20; i++ {
		str = strings.Replace(str, "{{{NGINX_PROJECT_PORT+"+strconv.Itoa(i)+"}}}", strconv.Itoa(portNumberRanged+17000+i), -1)
	}
	str = strings.Replace(str, "{{{NETWORK_NUMBER}}}", strconv.Itoa(portNumber+90), -1)
	str = strings.Replace(str, "{{{PROJECT_NAME}}}", projectName, -1)

	resultFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName) + "/docker-compose.yml"
	err = ioutil.WriteFile(resultFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	b, err = os.ReadFile(dockerDefFileForOS)
	if err != nil {
		log.Fatal(err)
	}

	str = string(b)
	portsConfig = configs.ParseFile(paths.GetExecDirPath() + "/aruntime/ports.conf")
	portNumber, err = strconv.Atoi(portsConfig[projectName])
	if err != nil {
		log.Fatal(err)
	}

	portNumberRanged = (portNumber - 1) * 20
	hostName = "loc." + projectName + ".com"
	projectConf = configs.GetCurrentProjectConfig()
	if val, ok := projectConf["HOSTS"]; ok {
		hosts := strings.Split(val, " ")
		if len(hosts) > 0 {
			hostName = strings.Split(hosts[0], ":")[0]
		}
	}
	str = configs.ReplaceConfigValue(str)
	str = strings.Replace(str, "{{{HOST_NAME_DEFAULT}}}", hostName, -1)
	str = strings.Replace(str, "{{{NGINX_PROJECT_PORT}}}", strconv.Itoa(portNumberRanged+17000), -1)
	str = strings.Replace(str, "{{{NGINX_PROJECT_PORT_SSL}}}", strconv.Itoa(portNumberRanged+17001), -1)
	for i := 2; i < 20; i++ {
		str = strings.Replace(str, "{{{NGINX_PROJECT_PORT+"+strconv.Itoa(i)+"}}}", strconv.Itoa(portNumberRanged+17000+i), -1)
	}
	str = strings.Replace(str, "{{{NETWORK_NUMBER}}}", strconv.Itoa(portNumber+90), -1)

	resultFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName) + "/docker-compose.override.yml"
	err = ioutil.WriteFile(resultFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeDBDockerfile(projectName string) {
	dockerDefFile := getDockerConfigFile(projectName, "/docker/db/Dockerfile")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs.ReplaceConfigValue(str)
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/db.Dockerfile"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	myCnfFile := getDockerConfigFile(projectName, "/docker/db/my.cnf")
	if _, err := os.Stat(myCnfFile); os.IsNotExist(err) {
		log.Fatal(err)
	}

	b, err = os.ReadFile(myCnfFile)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx/my.cnf", b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeElasticDockerfile(projectName string) {
	dockerDefFile := getDockerConfigFile(projectName, "/docker/elasticsearch/Dockerfile")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs.ReplaceConfigValue(str)
	nginxFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/elasticsearch.Dockerfile"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeRedisDockerfile(projectName string) {
	dockerDefFile := getDockerConfigFile(projectName, "/docker/redis/Dockerfile")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs.ReplaceConfigValue(str)
	nginxFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/redis.Dockerfile"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeNodeDockerfile(projectName string) {
	dockerDefFile := getDockerConfigFile(projectName, "/docker/node/Dockerfile")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs.ReplaceConfigValue(str)
	nginxFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/node.Dockerfile"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func getDockerConfigFile(projectName, path string) string {
	dockerDefFile := paths.GetExecDirPath() + "/projects/" + projectName + path
	if _, err := os.Stat(dockerDefFile); os.IsNotExist(err) {
		dockerDefFile = paths.GetExecDirPath() + path
		if _, err = os.Stat(dockerDefFile); os.IsNotExist(err) {
			log.Fatal(err)
		}
	}

	return dockerDefFile
}
