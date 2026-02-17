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

	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/dockertransform"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/faradey/madock/v3/src/helper/ports"
)

func MakeConf(projectName string) {
	if paths.IsFileExist(paths.CacheDir() + "/conf-cache") {
		return
	}
	// get project config
	projectConf := configs.GetProjectConfig(projectName)
	pp := paths.NewProjectPaths(projectName)
	src := paths.MakeDirsByPath(pp.RuntimeDir()) + "/src"
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
	if gen, ok := dockerConfGenerators[projectConf["platform"]]; ok {
		gen(projectName)
	}
	processOtherCTXFiles(projectName)
}

func MakeScriptsConf(projectName string) {
	exPath := paths.GetExecDirPath()
	pp := paths.NewProjectPaths(projectName)
	src := pp.CtxDir() + "/scripts"
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

func MakeKibanaConf(projectName string) {
	file := GetDockerConfigFile(projectName, "kibana/kibana.yml", "")

	b, err := os.ReadFile(file)
	if err != nil {
		logger.Fatal(err)
	}

	b = ProcessSnippets(b, projectName)
	str := string(b)
	str = configs.ReplaceConfigValue(projectName, str)

	pp := paths.NewProjectPaths(projectName)
	filePath := paths.MakeDirsByPath(pp.CtxDir()) + "/kibana.yml"
	err = os.WriteFile(filePath, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeNginxDockerfile(projectName string) {
	MakeDockerfile(projectName, "nginx/Dockerfile", "nginx.Dockerfile")
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

	// Replace main_service placeholder for proxy-based configs
	mainService := resolveMainService(projectConf)
	str = strings.Replace(str, "{{{main_service}}}", mainService, -1)

	pp := paths.NewProjectPaths(projectName)
	paths.MakeDirsByPath(pp.CtxDir())
	nginxFile := paths.MakeDirsByPath(pp.CtxDir()) + "/nginx.conf"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func MakePhpDockerfile(projectName string) {
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
	str = dockertransform.ApplyDockerfileTransform("php.Dockerfile", str)
	pp := paths.NewProjectPaths(projectName)
	nginxFile := paths.MakeDirsByPath(pp.CtxDir()) + "/php.Dockerfile"
	err = os.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	dockerDefFileWithoutXdebug := GetDockerConfigFileOptional(projectName, "php/DockerfileWithoutXdebug", "")
	if dockerDefFileWithoutXdebug != "" {
		b, err = os.ReadFile(dockerDefFileWithoutXdebug)
		if err != nil {
			logger.Fatal(err)
		}

		b = ProcessSnippets(b, projectName)
		str = string(b)
		str = configs.ReplaceConfigValue(projectName, str)
		str = dockertransform.ApplyDockerfileTransform("php.DockerfileWithoutXdebug", str)
		nginxFile = paths.MakeDirsByPath(pp.CtxDir()) + "/php.DockerfileWithoutXdebug"
		err = os.WriteFile(nginxFile, []byte(str), 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}
	}
}

func MakeMainContainerDockerfile(projectName string) {
	projectConf := configs.GetProjectConfig(projectName)
	language := projectConf["language"]
	if language == "" {
		language = "php"
	}

	switch language {
	case "php":
		makeCustomPhpDockerfile(projectName)
	case "nodejs":
		MakeDockerfile(projectName, "Dockerfile", "nodejs.Dockerfile")
	case "python":
		MakeDockerfile(projectName, "Dockerfile", "python.Dockerfile")
	case "golang":
		MakeDockerfile(projectName, "Dockerfile", "golang.Dockerfile")
	case "ruby":
		MakeDockerfile(projectName, "Dockerfile", "ruby.Dockerfile")
	case "none":
		MakeDockerfile(projectName, "Dockerfile", "app.Dockerfile")
	default:
		makeCustomPhpDockerfile(projectName)
	}
}

func makeCustomPhpDockerfile(projectName string) {
	dockerDefFile := GetDockerConfigFile(projectName, "Dockerfile", "")

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
	str = dockertransform.ApplyDockerfileTransform("php.Dockerfile", str)
	pp := paths.NewProjectPaths(projectName)
	phpFile := paths.MakeDirsByPath(pp.CtxDir()) + "/php.Dockerfile"
	err = os.WriteFile(phpFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	dockerDefFileWithoutXdebug := GetDockerConfigFileOptional(projectName, "DockerfileWithoutXdebug", "")
	if dockerDefFileWithoutXdebug != "" {
		b, err = os.ReadFile(dockerDefFileWithoutXdebug)
		if err != nil {
			logger.Fatal(err)
		}

		b = ProcessSnippets(b, projectName)
		str = string(b)
		str = configs.ReplaceConfigValue(projectName, str)
		str = dockertransform.ApplyDockerfileTransform("php.DockerfileWithoutXdebug", str)
		phpFile = paths.MakeDirsByPath(pp.CtxDir()) + "/php.DockerfileWithoutXdebug"
		err = os.WriteFile(phpFile, []byte(str), 0755)
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

		hostName := "loc." + projectName + ".com"
		hosts := configs.GetHosts(projectConf)
		if len(hosts) > 0 {
			hostName = hosts[0]["name"]
		}
		str = configs.ReplaceConfigValue(projectName, str)
		str = strings.Replace(str, "{{{nginx/host_name_default}}}", hostName, -1)

		// Replace main_service placeholder for nginx depends_on
		mainService := resolveMainService(projectConf)
		str = strings.Replace(str, "{{{main_service}}}", mainService, -1)

		// Dynamic port placeholder replacement - scans for any {{{port/XXX}}} pattern
		str = replacePortPlaceholders(str, projectName)

		str = strings.Replace(str, "{{{project_name}}}", strings.ToLower(projectName), -1)
		str = strings.Replace(str, "{{{scope}}}", configs.GetActiveScope(projectName, false, "-"), -1)

		str = dockertransform.ApplyComposeTransform(key, str)

		pp := paths.NewProjectPaths(projectName)
		resultFile := paths.MakeDirsByPath(pp.RuntimeDir()) + "/" + key
		err = os.WriteFile(resultFile, []byte(str), 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}
	}
}

// resolveMainService determines the main service name based on the language config
func resolveMainService(projectConf map[string]string) string {
	language := projectConf["language"]
	switch language {
	case "nodejs":
		return "nodejs"
	case "python":
		return "python"
	case "golang":
		return "golang"
	case "ruby":
		return "ruby"
	case "none":
		return "app"
	default:
		return "php"
	}
}

// replacePortPlaceholders dynamically scans for {{{port/XXX}}} patterns and allocates ports
func replacePortPlaceholders(str, projectName string) string {
	re := regexp.MustCompile(`\{\{\{port/([a-z0-9_]+)\}\}\}`)
	matches := re.FindAllStringSubmatch(str, -1)

	// Use a map to avoid duplicate replacements
	replaced := make(map[string]bool)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		placeholder := match[0]
		serviceName := match[1]

		if replaced[placeholder] {
			continue
		}
		replaced[placeholder] = true

		port := ports.GetPort(projectName, serviceName)
		str = strings.Replace(str, placeholder, strconv.Itoa(port), -1)
	}

	return str
}

func MakeDBDockerfile(projectName string) {
	dockerDefFile := GetDockerConfigFile(projectName, "/db/Dockerfile", "")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		logger.Fatal(err)
	}

	b = ProcessSnippets(b, projectName)
	str := string(b)
	str = configs.ReplaceConfigValue(projectName, str)
	str = dockertransform.ApplyDockerfileTransform("db.Dockerfile", str)
	pp := paths.NewProjectPaths(projectName)
	nginxFile := paths.MakeDirsByPath(pp.CtxDir()) + "/db.Dockerfile"
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

	if strings.ToLower(configs.GetProjectConfig(projectName)["db/repository"]) == "mariadb" && configs.CompareVersions(configs.GetProjectConfig(projectName)["db/version"], "10.4") >= 0 {
		b = bytes.Replace(b, []byte("[mysqld]"), []byte("[mysqld]\noptimizer_switch = 'rowid_filter=off'\noptimizer_use_condition_selectivity = 1\n"), -1)
	}

	err = os.WriteFile(pp.CtxDir()+"/my.cnf", b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func MakeElasticDockerfile(projectName string) {
	MakeDockerfile(projectName, "elasticsearch/Dockerfile", "elasticsearch.Dockerfile")
}

func MakeOpenSearchDockerfile(projectName string) {
	MakeDockerfile(projectName, "opensearch/Dockerfile", "opensearch.Dockerfile")
}

func MakeRedisDockerfile(projectName string) {
	MakeDockerfile(projectName, "redis/Dockerfile", "redis.Dockerfile")
}

func MakeNodeJsDockerfile(projectName string) {
	MakeDockerfile(projectName, "nodejs/Dockerfile", "nodejs.Dockerfile")
}

func MakeClaudeDockerfile(projectName string) {
	MakeDockerfile(projectName, "claude/Dockerfile", "claude.Dockerfile")
}

func MakeDockerfile(projectName, path, fileName string) {
	dockerDefFile := GetDockerConfigFile(projectName, path, "")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		logger.Fatal(err)
	}

	b = ProcessSnippets(b, projectName)
	str := string(b)
	str = configs.ReplaceConfigValue(projectName, str)
	str = dockertransform.ApplyDockerfileTransform(fileName, str)

	pp := paths.NewProjectPaths(projectName)
	dockerFile := paths.MakeDirsByPath(pp.CtxDir()) + "/" + fileName
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
	language := projectConf["language"]
	dockerDefFile := paths.GetRunDirPath() + "/.madock/docker/" + strings.Trim(path, "/")
	if !paths.IsFileExist(dockerDefFile) {
		dockerDefFile = paths.GetExecDirPath() + "/projects/" + projectName + "/docker/" + strings.Trim(path, "/")
		if !paths.IsFileExist(dockerDefFile) {
			dockerDefFile = paths.GetExecDirPath() + "/docker/" + platform + "/" + strings.Trim(path, "/")
			if !paths.IsFileExist(dockerDefFile) {
				// Language-specific fallback (for all languages on custom platform)
				if language != "" {
					dockerDefFile = paths.GetExecDirPath() + "/docker/languages/" + language + "/" + strings.Trim(path, "/")
				}
				if !paths.IsFileExist(dockerDefFile) {
					dockerDefFile = paths.GetExecDirPath() + "/docker/general/service/" + strings.Trim(path, "/")
					if !paths.IsFileExist(dockerDefFile) {
						logger.Fatal(fmt.Errorf("docker config file not found: %s (platform=%s, language=%s)", path, platform, language))
					}
				}
			}
		}
	}

	return dockerDefFile
}

func GetDockerConfigFileOptional(projectName, path, platform string) string {
	projectConf := configs.GetProjectConfig(projectName)
	if platform == "" {
		platform = projectConf["platform"]
	}
	language := projectConf["language"]
	dockerDefFile := paths.GetRunDirPath() + "/.madock/docker/" + strings.Trim(path, "/")
	if !paths.IsFileExist(dockerDefFile) {
		dockerDefFile = paths.GetExecDirPath() + "/projects/" + projectName + "/docker/" + strings.Trim(path, "/")
		if !paths.IsFileExist(dockerDefFile) {
			dockerDefFile = paths.GetExecDirPath() + "/docker/" + platform + "/" + strings.Trim(path, "/")
			if !paths.IsFileExist(dockerDefFile) {
				if language != "" {
					dockerDefFile = paths.GetExecDirPath() + "/docker/languages/" + language + "/" + strings.Trim(path, "/")
				}
				if !paths.IsFileExist(dockerDefFile) {
					dockerDefFile = paths.GetExecDirPath() + "/docker/general/service/" + strings.Trim(path, "/")
					if !paths.IsFileExist(dockerDefFile) {
						return ""
					}
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
		pp := paths.NewProjectPaths(projectName)
		paths.MakeDirsByPath(pp.CtxDir() + "/" + strings.Split(fileName, "/")[0] + "/")
		destinationFile := pp.CtxDir() + "/" + fileName
		err = os.WriteFile(destinationFile, []byte(str), 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}
	}

	pp := paths.NewProjectPaths(projectName)
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/docker/ctx/")
	ctxFiles := paths.GetFiles(paths.GetExecDirPath() + "/projects/" + projectName + "/docker/ctx/")
	for _, ctxFile := range ctxFiles {
		b, err = os.ReadFile(paths.GetExecDirPath() + "/projects/" + projectName + "/docker/ctx/" + ctxFile)
		if err != nil {
			logger.Fatal(err)
		}
		b = ProcessSnippets(b, projectName)
		str := string(b)
		destinationFile := pp.CtxDir() + "/" + ctxFile
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
