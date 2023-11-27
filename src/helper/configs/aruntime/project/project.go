package project

import (
	"bytes"
	"fmt"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func MakeConf(projectName string) {
	// get project config
	projectConf := configs2.GetProjectConfig(projectName)
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

	makeNginxDockerfile(projectName)
	makeNginxConf(projectName)
	makeDockerCompose(projectName)
	if projectConf["PLATFORM"] == "magento2" {
		MakeConfMagento2(projectName)
	} else if projectConf["PLATFORM"] == "pwa" {
		MakeConfPWA(projectName)
	} else if projectConf["PLATFORM"] == "shopify" {
		MakeConfShopify(projectName)
	} else if projectConf["PLATFORM"] == "custom" {
		MakeConfCustom(projectName)
	}
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
	file := GetDockerConfigFile(projectName, "kibana/kibana.yml", "")

	b, err := os.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	str := string(b)
	str = configs2.ReplaceConfigValue(str)

	filePath := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/kibana.yml"
	err = os.WriteFile(filePath, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeNginxDockerfile(projectName string) {
	dockerDefFile := GetDockerConfigFile(projectName, "nginx/Dockerfile", "")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}
	str := string(b)
	str = configs2.ReplaceConfigValue(str)

	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/nginx.Dockerfile"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeNginxConf(projectName string) {
	projectConf := configs2.GetCurrentProjectConfig()
	defFile := GetDockerConfigFile(projectName, "nginx/conf/default.conf", "")

	b, err := os.ReadFile(defFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs2.ReplaceConfigValue(str)
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
	str = strings.Replace(str, "{{{PROJECT_NAME}}}", strings.ToLower(projectName), -1)
	str = strings.Replace(str, "{{{HOST_NAMES_WEBSITES}}}", hostNameWebsites, -1)

	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx")
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/nginx.conf"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makePhpDockerfile(projectName string) {
	dockerDefFile := GetDockerConfigFile(projectName, "php/Dockerfile", "")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs2.ReplaceConfigValue(str)
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/php.Dockerfile"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	projectConf := configs2.GetCurrentProjectConfig()
	if _, err := os.Stat(paths.GetExecDirPath() + "/docker/" + projectConf["PLATFORM"] + "/php/DockerfileWithoutXdebug"); !os.IsNotExist(err) {
		dockerDefFile = GetDockerConfigFile(projectName, "php/DockerfileWithoutXdebug", "")
		b, err = os.ReadFile(dockerDefFile)
		if err != nil {
			log.Fatal(err)
		}

		str = string(b)
		str = configs2.ReplaceConfigValue(str)
		nginxFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/php.DockerfileWithoutXdebug"
		err = os.WriteFile(nginxFile, []byte(str), 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}
	}
}

func makeDockerCompose(projectName string) {
	overrideFile := runtime.GOOS
	projectConf := configs2.GetCurrentProjectConfig()

	dockerDefFile := GetDockerConfigFile(projectName, "docker-compose.yml", "")
	dockerDefFileForOS := GetDockerConfigFile(projectName, "docker-compose."+overrideFile+".yml", "")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	portsConfig := configs2.ParseFile(paths.GetExecDirPath() + "/aruntime/ports.conf")
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
	str = configs2.ReplaceConfigValue(str)
	str = strings.Replace(str, "{{{HOST_NAME_DEFAULT}}}", hostName, -1)
	str = strings.Replace(str, "{{{NGINX_PROJECT_PORT}}}", strconv.Itoa(portNumberRanged+17000), -1)
	str = strings.Replace(str, "{{{NGINX_PROJECT_PORT_SSL}}}", strconv.Itoa(portNumberRanged+17001), -1)
	for i := 2; i < 20; i++ {
		str = strings.Replace(str, "{{{NGINX_PROJECT_PORT+"+strconv.Itoa(i)+"}}}", strconv.Itoa(portNumberRanged+17000+i), -1)
	}
	str = strings.Replace(str, "{{{NETWORK_NUMBER}}}", strconv.Itoa(portNumber+90), -1)
	str = strings.Replace(str, "{{{PROJECT_NAME}}}", strings.ToLower(projectName), -1)

	resultFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName) + "/docker-compose.yml"
	err = os.WriteFile(resultFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	b, err = os.ReadFile(dockerDefFileForOS)
	if err != nil {
		log.Fatal(err)
	}

	str = string(b)
	portsConfig = configs2.ParseFile(paths.GetExecDirPath() + "/aruntime/ports.conf")
	portNumber, err = strconv.Atoi(portsConfig[projectName])
	if err != nil {
		log.Fatal(err)
	}

	portNumberRanged = (portNumber - 1) * 20
	hostName = "loc." + projectName + ".com"
	projectConf = configs2.GetCurrentProjectConfig()
	if val, ok := projectConf["HOSTS"]; ok {
		hosts := strings.Split(val, " ")
		if len(hosts) > 0 {
			hostName = strings.Split(hosts[0], ":")[0]
		}
	}
	str = configs2.ReplaceConfigValue(str)
	str = strings.Replace(str, "{{{HOST_NAME_DEFAULT}}}", hostName, -1)
	str = strings.Replace(str, "{{{NGINX_PROJECT_PORT}}}", strconv.Itoa(portNumberRanged+17000), -1)
	str = strings.Replace(str, "{{{NGINX_PROJECT_PORT_SSL}}}", strconv.Itoa(portNumberRanged+17001), -1)
	for i := 2; i < 20; i++ {
		str = strings.Replace(str, "{{{NGINX_PROJECT_PORT+"+strconv.Itoa(i)+"}}}", strconv.Itoa(portNumberRanged+17000+i), -1)
	}
	str = strings.Replace(str, "{{{NETWORK_NUMBER}}}", strconv.Itoa(portNumber+90), -1)

	resultFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName) + "/docker-compose.override.yml"
	err = os.WriteFile(resultFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeDBDockerfile(projectName string) {
	dockerDefFile := GetDockerConfigFile(projectName, "/db/Dockerfile", "")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs2.ReplaceConfigValue(str)
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/db.Dockerfile"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	myCnfFile := GetDockerConfigFile(projectName, "db/my.cnf", "")
	if _, err := os.Stat(myCnfFile); os.IsNotExist(err) {
		log.Fatal(err)
	}

	b, err = os.ReadFile(myCnfFile)
	if err != nil {
		log.Fatal(err)
	}

	if strings.ToLower(configs2.GetCurrentProjectConfig()["DB_REPOSITORY"]) == "mariadb" && configs2.GetCurrentProjectConfig()["DB_VERSION"] >= "10.4" {
		b = bytes.Replace(b, []byte("[mysqld]"), []byte("[mysqld]\noptimizer_switch = 'rowid_filter=off'\noptimizer_use_condition_selectivity = 1\n"), -1)
	}

	err = os.WriteFile(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx/my.cnf", b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeElasticDockerfile(projectName string) {
	dockerDefFile := GetDockerConfigFile(projectName, "elasticsearch/Dockerfile", "")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs2.ReplaceConfigValue(str)
	nginxFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/elasticsearch.Dockerfile"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeOpenSearchDockerfile(projectName string) {
	dockerDefFile := GetDockerConfigFile(projectName, "opensearch/Dockerfile", "")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs2.ReplaceConfigValue(str)
	nginxFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/opensearch.Dockerfile"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeRedisDockerfile(projectName string) {
	dockerDefFile := GetDockerConfigFile(projectName, "redis/Dockerfile", "")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs2.ReplaceConfigValue(str)
	nginxFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/redis.Dockerfile"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeNodeJsDockerfile(projectName string) {
	dockerDefFile := GetDockerConfigFile(projectName, "nodejs/Dockerfile", "")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs2.ReplaceConfigValue(str)
	nodeJsFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/nodejs.Dockerfile"
	err = os.WriteFile(nodeJsFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func GetDockerConfigFile(projectName, path, platform string) string {
	projectConf := configs2.GetCurrentProjectConfig()
	if platform == "" {
		platform = projectConf["PLATFORM"]
	}
	dockerDefFile := paths.GetExecDirPath() + "/projects/" + projectName + "/docker/" + strings.Trim(path, "/")
	if _, err := os.Stat(dockerDefFile); os.IsNotExist(err) {
		dockerDefFile = paths.GetExecDirPath() + "/docker/" + platform + "/" + strings.Trim(path, "/")
		if _, err = os.Stat(dockerDefFile); os.IsNotExist(err) {
			log.Fatal(err)
		}
	}

	return dockerDefFile
}

func processOtherCTXFiles(projectName string) {
	filesNames := []string{
		/*"mftf/mftf_runner.sh",*/
	}
	var b []byte
	var err error
	var file string
	for _, fileName := range filesNames {
		file = GetDockerConfigFile(projectName, fileName, "")
		b, err = os.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}

		str := string(b)
		str = configs2.ReplaceConfigValue(str)
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/" + strings.Split(fileName, "/")[0] + "/")
		destinationFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/" + fileName
		err = os.WriteFile(destinationFile, []byte(str), 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}
	}
}
