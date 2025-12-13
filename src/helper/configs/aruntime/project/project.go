package project

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	configs2 "github.com/faradey/madock/src/migration/versions/v240/configs"
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
	err := os.Symlink(projectConf["path"], src)
	if err != nil {
		logger.Fatal(err)
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
	} else if projectConf["platform"] == "shopware" {
		MakeConfShopware(projectName)
	} else if projectConf["platform"] == "prestashop" {
		MakeConfPrestashop(projectName)
	}
	processOtherCTXFiles(projectName)
}

func makeScriptsConf(projectName string) {
	exPath := paths.GetExecDirPath()
	src := exPath + "/aruntime/projects/" + projectName + "/ctx/scripts"
	if fi, err := os.Lstat(src); err == nil {
		if fi.Mode()&os.ModeSymlink != os.ModeSymlink {
			err = os.RemoveAll(src)
			if err == nil {
				err = os.Symlink(exPath+"/scripts", src)
				if err != nil {
					logger.Fatal(err)
				}
			} else {
				fmt.Println(err)
			}
		}
	} else {
		err = os.Symlink(exPath+"/scripts", src)
		if err != nil {
			logger.Fatal(err)
		}
	}
}

func makeKibanaConf(projectName string) {
	file := GetDockerConfigFile(projectName, "kibana/kibana.yml", "")

	b, err := os.ReadFile(file)
	if err != nil {
		logger.Fatal(err)
	}

	b = ProcessSnippets(b, projectName)
	str := string(b)
	str = configs.ReplaceConfigValue(projectName, str)

	filePath := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/kibana.yml"
	err = os.WriteFile(filePath, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeNginxDockerfile(projectName string) {
	makeDockerfile(projectName, "nginx/Dockerfile", "nginx.Dockerfile")
}

func makeNginxConf(projectName string) {
	projectConf := configs.GetProjectConfig(projectName)
	defFile := GetDockerConfigFile(projectName, "nginx/conf/default.conf", "")

	b, err := os.ReadFile(defFile)
	if err != nil {
		logger.Fatal(err)
	}

	b = ProcessSnippets(b, projectName)
	str := string(b)
	str = configs.ReplaceConfigValue(projectName, str)
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
		logger.Fatal(err)
	}
	projectConf := configs.GetProjectConfig(projectName)
	nodeMajorVersion := strings.Split(projectConf["nodejs/version"], ".")
	if len(nodeMajorVersion) > 0 {
		projectConf["nodejs/major_version"] = nodeMajorVersion[0]
	}

	b = ProcessSnippets(b, projectName)
	str := string(b)
	str = configs.ReplaceConfigValue(projectName, str)
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/php.Dockerfile"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	if paths.IsFileExist(paths.GetExecDirPath() + "/docker/" + projectConf["platform"] + "/php/DockerfileWithoutXdebug") {
		dockerDefFile = GetDockerConfigFile(projectName, "php/DockerfileWithoutXdebug", "")
		b, err = os.ReadFile(dockerDefFile)
		if err != nil {
			logger.Fatal(err)
		}

		b = ProcessSnippets(b, projectName)
		str = string(b)
		str = configs.ReplaceConfigValue(projectName, str)
		nginxFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/php.DockerfileWithoutXdebug"
		err = os.WriteFile(nginxFile, []byte(str), 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}
	}
}

func makeDockerCompose(projectName string) {
	overrideFile := runtime.GOOS
	projectConf := configs.GetProjectConfig(projectName)
	var dockerDefFiles map[string]string
	dockerDefFiles = make(map[string]string)
	dockerDefFiles["docker-compose.yml"] = GetDockerConfigFile(projectName, "docker-compose.yml", "")
	dockerDefFiles["docker-compose.override.yml"] = GetDockerConfigFile(projectName, "docker-compose."+overrideFile+".yml", "")
	dockerDefFiles["docker-compose-snapshot.yml"] = GetDockerConfigFile(projectName, "docker-compose-snapshot.yml", "general")
	for key, dockerDefFile := range dockerDefFiles {
		b, err := os.ReadFile(dockerDefFile)
		if err != nil {
			logger.Fatal(err)
		}
		b = ProcessSnippets(b, projectName)

		str := string(b)
		portsConfig := configs2.ParseFile(paths.GetExecDirPath() + "/aruntime/ports.conf")
		portNumber, err := strconv.Atoi(portsConfig[projectName])
		if err != nil {
			logger.Fatal(err)
		}

		portNumberRanged := (portNumber - 1) * 12
		hostName := "loc." + projectName + ".com"
		hosts := configs.GetHosts(projectConf)
		if len(hosts) > 0 {
			hostName = hosts[0]["name"]
		}
		str = configs.ReplaceConfigValue(projectName, str)
		str = strings.Replace(str, "{{{nginx/host_name_default}}}", hostName, -1)
		str = strings.Replace(str, "{{{nginx/port/project}}}", strconv.Itoa(portNumberRanged+17000), -1)
		str = strings.Replace(str, "{{{nginx/port/project_ssl}}}", strconv.Itoa(portNumberRanged+17001), -1)
		for i := 2; i < 12; i++ {
			str = strings.Replace(str, "{{{nginx/port/project+"+strconv.Itoa(i)+"}}}", strconv.Itoa(portNumberRanged+17000+i), -1)
		}
		str = strings.Replace(str, "{{{project_name}}}", strings.ToLower(projectName), -1)
		str = strings.Replace(str, "{{{scope}}}", configs.GetActiveScope(projectName, false, "-"), -1)

		resultFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName) + "/" + key
		err = os.WriteFile(resultFile, []byte(str), 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}
	}
}

func CompareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLength := len(parts1)
	if len(parts2) > maxLength {
		maxLength = len(parts2)
	}

	for i := 0; i < maxLength; i++ {
		var p1, p2 int

		if i < len(parts1) {
			p1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			p2, _ = strconv.Atoi(parts2[i])
		}

		if p1 > p2 {
			return 1
		} else if p1 < p2 {
			return -1
		}
	}

	return 0
}

func makeDBDockerfile(projectName string) {
	dockerDefFile := GetDockerConfigFile(projectName, "/db/Dockerfile", "")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		logger.Fatal(err)
	}

	b = ProcessSnippets(b, projectName)
	str := string(b)
	str = configs.ReplaceConfigValue(projectName, str)
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/db.Dockerfile"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	myCnfFile := GetDockerConfigFile(projectName, "db/my.cnf", "")
	if !paths.IsFileExist(myCnfFile) {
		logger.Fatal(err)
	}

	b, err = os.ReadFile(myCnfFile)
	if err != nil {
		logger.Fatal(err)
	}
	b = ProcessSnippets(b, projectName)

	if strings.ToLower(configs.GetProjectConfig(projectName)["db/repository"]) == "mariadb" && CompareVersions(configs.GetProjectConfig(projectName)["db/version"], "10.4") >= 0 {
		b = bytes.Replace(b, []byte("[mysqld]"), []byte("[mysqld]\noptimizer_switch = 'rowid_filter=off'\noptimizer_use_condition_selectivity = 1\n"), -1)
	}

	err = os.WriteFile(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx/my.cnf", b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeElasticDockerfile(projectName string) {
	makeDockerfile(projectName, "elasticsearch/Dockerfile", "elasticsearch.Dockerfile")
}

func makeOpenSearchDockerfile(projectName string) {
	makeDockerfile(projectName, "opensearch/Dockerfile", "opensearch.Dockerfile")
}

func makeRedisDockerfile(projectName string) {
	makeDockerfile(projectName, "redis/Dockerfile", "redis.Dockerfile")
}

func makeNodeJsDockerfile(projectName string) {
	makeDockerfile(projectName, "nodejs/Dockerfile", "nodejs.Dockerfile")
}

func makeClaudeDockerfile(projectName string) {
	makeDockerfile(projectName, "claude/Dockerfile", "claude.Dockerfile")
}

func makeDockerfile(projectName, path, fileName string) {
	dockerDefFile := GetDockerConfigFile(projectName, path, "")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		logger.Fatal(err)
	}

	b = ProcessSnippets(b, projectName)
	str := string(b)
	str = configs.ReplaceConfigValue(projectName, str)
	dockerFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/" + fileName
	err = os.WriteFile(dockerFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func GetDockerConfigFile(projectName, path, platform string) string {
	projectConf := configs.GetProjectConfig(projectName)
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
				dockerDefFile = paths.GetExecDirPath() + "/docker/general/service/" + strings.Trim(path, "/")
				if !paths.IsFileExist(dockerDefFile) {
					logger.Fatal(err)
				}
			}
		}
	}

	return dockerDefFile
}

func processOtherCTXFiles(projectName string) {
	filesNames := []string{
		"grafana/loki-config.yaml",
		"grafana/promtail-config.yml",
		"grafana/prometheus-config.yml",
		"grafana/mysql-exporter.my.cnf",
		"grafana/dashboard-mysql.json",
		"grafana/dashboard-redis.json",
		"grafana/dashboard-loki.json",
	}
	var b []byte
	var err error
	var file string
	for _, fileName := range filesNames {
		file = GetDockerConfigFile(projectName, fileName, "")
		b, err = os.ReadFile(file)
		if err != nil {
			logger.Fatal(err)
		}

		b = ProcessSnippets(b, projectName)
		str := string(b)
		str = configs.ReplaceConfigValue(projectName, str)
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
			logger.Fatal(err)
		}
		b = ProcessSnippets(b, projectName)
		str := string(b)
		destinationFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/" + ctxFile
		err = os.WriteFile(destinationFile, []byte(str), 0755)
	}
}

func ProcessSnippets(b []byte, projectName string) []byte {
	str := string(b)
	r := regexp.MustCompile(`\{\{\{include snippets/[^\}]+\}\}\}`)

	for _, match := range r.FindAllString(str, -1) {
		snippetFile := strings.Replace(match, "{{{include ", "", -1)
		snippetFile = strings.TrimSpace(strings.Replace(snippetFile, "}}}", "", -1))
		snippetFile = GetSnippetFile(projectName, snippetFile)

		b2, err := os.ReadFile(snippetFile)
		if err != nil {
			logger.Fatal(err)
		}
		str = strings.Replace(str, match, string(b2), -1)
	}

	return []byte(str)
}

func GetSnippetFile(projectName, path string) string {
	snippetFile := paths.GetRunDirPath() + "/.madock/docker/" + strings.Trim(path, "/")
	if !paths.IsFileExist(snippetFile) {
		snippetFile = paths.GetExecDirPath() + "/projects/" + projectName + "/docker/" + strings.Trim(path, "/")
		if !paths.IsFileExist(snippetFile) {
			snippetFile = paths.GetExecDirPath() + "/docker/" + strings.Trim(path, "/")
			if !paths.IsFileExist(snippetFile) {
				logger.Fatal("The file " + path + " does not exist")
			}
		}
	}

	return snippetFile
}
