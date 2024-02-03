package project

import (
	"bytes"
	"fmt"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	configs2 "github.com/faradey/madock/src/migration/versions/v240/configs"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func MakeConf(projectName string) {
	if paths.IsFileExist(paths.GetExecDirPath() + "/cache/conf-cache") {
		return
	}
	// get project config
	projectConf := configs.GetProjectConfig(projectName)
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
	if projectConf["platform"] == "magento2" {
		MakeConfMagento2(projectName)
	} else if projectConf["platform"] == "pwa" {
		MakeConfPWA(projectName)
	} else if projectConf["platform"] == "shopify" {
		MakeConfShopify(projectName)
	} else if projectConf["platform"] == "custom" {
		MakeConfCustom(projectName)
	}
	processOtherCTXFiles(projectName)
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
	str = configs.ReplaceConfigValue(str)

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
	str = configs.ReplaceConfigValue(str)

	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/nginx.Dockerfile"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeNginxConf(projectName string) {
	projectConf := configs.GetCurrentProjectConfig()
	defFile := GetDockerConfigFile(projectName, "nginx/conf/default.conf", "")

	b, err := os.ReadFile(defFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs.ReplaceConfigValue(str)
	hostName := "loc." + projectName + ".com"
	hostNameWebsites := "loc." + projectName + ".com base;"
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		var onlyHosts []string
		var websitesHosts []string
		for _, host := range hosts {
			websitesHosts = append(websitesHosts, host["name"]+" "+host["code"]+";")
			onlyHosts = append(onlyHosts, host["name"])
		}
		if len(onlyHosts) > 0 {
			hostName = strings.Join(onlyHosts, "\n")
		}
		if len(websitesHosts) > 0 {
			hostNameWebsites = strings.Join(websitesHosts, "\n")
		}
	}
	str = strings.Replace(str, "{{{nginx/host_names}}}", hostName, -1)
	str = strings.Replace(str, "{{{project_name}}}", strings.ToLower(projectName), -1)

	str = strings.Replace(str, "{{{scope}}}", configs.GetActiveScope(projectName, false, "-"), -1)
	str = strings.Replace(str, "{{{nginx/host_names_with_codes}}}", hostNameWebsites, -1)

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
	str = configs.ReplaceConfigValue(str)
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/php.Dockerfile"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	projectConf := configs.GetCurrentProjectConfig()
	if paths.IsFileExist(paths.GetExecDirPath() + "/docker/" + projectConf["platform"] + "/php/DockerfileWithoutXdebug") {
		dockerDefFile = GetDockerConfigFile(projectName, "php/DockerfileWithoutXdebug", "")
		b, err = os.ReadFile(dockerDefFile)
		if err != nil {
			log.Fatal(err)
		}

		str = string(b)
		str = configs.ReplaceConfigValue(str)
		nginxFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/php.DockerfileWithoutXdebug"
		err = os.WriteFile(nginxFile, []byte(str), 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}
	}
}

func makeDockerCompose(projectName string) {
	overrideFile := runtime.GOOS
	projectConf := configs.GetCurrentProjectConfig()

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
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		hostName = hosts[0]["name"]
	}
	str = configs.ReplaceConfigValue(str)
	str = strings.Replace(str, "{{{nginx/host_name_default}}}", hostName, -1)
	str = strings.Replace(str, "{{{nginx/port/project}}}", strconv.Itoa(portNumberRanged+17000), -1)
	str = strings.Replace(str, "{{{nginx/port/project_ssl}}}", strconv.Itoa(portNumberRanged+17001), -1)
	for i := 2; i < 20; i++ {
		str = strings.Replace(str, "{{{nginx/port/project+"+strconv.Itoa(i)+"}}}", strconv.Itoa(portNumberRanged+17000+i), -1)
	}
	str = strings.Replace(str, "{{{nginx/network_number}}}", strconv.Itoa(portNumber+90), -1)
	str = strings.Replace(str, "{{{project_name}}}", strings.ToLower(projectName), -1)
	str = strings.Replace(str, "{{{scope}}}", configs.GetActiveScope(projectName, false, "-"), -1)

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
	projectConf = configs.GetCurrentProjectConfig()

	hosts = configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		hostName = hosts[0]["name"]
	}
	str = configs.ReplaceConfigValue(str)
	str = strings.Replace(str, "{{{nginx/host_name_default}}}", hostName, -1)
	str = strings.Replace(str, "{{{nginx/port/project}}}", strconv.Itoa(portNumberRanged+17000), -1)
	str = strings.Replace(str, "{{{nginx/port/project_ssl}}}", strconv.Itoa(portNumberRanged+17001), -1)
	for i := 2; i < 20; i++ {
		str = strings.Replace(str, "{{{nginx/port/project+"+strconv.Itoa(i)+"}}}", strconv.Itoa(portNumberRanged+17000+i), -1)
	}
	str = strings.Replace(str, "{{{nginx/network_number}}}", strconv.Itoa(portNumber+90), -1)

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
	str = configs.ReplaceConfigValue(str)
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/db.Dockerfile"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	myCnfFile := GetDockerConfigFile(projectName, "db/my.cnf", "")
	if !paths.IsFileExist(myCnfFile) {
		log.Fatal(err)
	}

	b, err = os.ReadFile(myCnfFile)
	if err != nil {
		log.Fatal(err)
	}

	if strings.ToLower(configs.GetCurrentProjectConfig()["db/repository"]) == "mariadb" && configs.GetCurrentProjectConfig()["db/version"] >= "10.4" {
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
	str = configs.ReplaceConfigValue(str)
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
	str = configs.ReplaceConfigValue(str)
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
	str = configs.ReplaceConfigValue(str)
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
	str = configs.ReplaceConfigValue(str)
	nodeJsFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/nodejs.Dockerfile"
	err = os.WriteFile(nodeJsFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func GetDockerConfigFile(projectName, path, platform string) string {
	projectConf := configs.GetCurrentProjectConfig()
	if platform == "" {
		platform = projectConf["platform"]
	}
	var err error
	dockerDefFile := paths.GetRunDirPath() + "/.madock/docker/" + strings.Trim(path, "/")
	if !paths.IsFileExist(dockerDefFile) {
		dockerDefFile = paths.GetExecDirPath() + "/projects/" + projectName + "/docker/" + strings.Trim(path, "/")
		if !paths.IsFileExist(dockerDefFile) {
			dockerDefFile = paths.GetExecDirPath() + "/docker/" + platform + "/" + strings.Trim(path, "/")
			if !paths.IsFileExist(dockerDefFile) {
				log.Fatal(err)
			}
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
		str = configs.ReplaceConfigValue(str)
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/" + strings.Split(fileName, "/")[0] + "/")
		destinationFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/" + fileName
		err = os.WriteFile(destinationFile, []byte(str), 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}
	}

	paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/docker/ctx/")
	ctxFiles := paths.GetFiles(paths.GetExecDirPath() + "/projects/" + projectName + "/docker/ctx/")
	for _, ctxFile := range ctxFiles {
		b, err = os.ReadFile(paths.GetExecDirPath() + "/projects/" + projectName + "/docker/ctx/" + ctxFile)
		if err != nil {
			log.Fatal(err)
		}
		str := string(b)
		destinationFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/" + ctxFile
		err = os.WriteFile(destinationFile, []byte(str), 0755)
	}
}
