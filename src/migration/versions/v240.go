package versions

import (
	"bytes"
	_ "embed"
	"encoding/xml"
	config2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/faradey/madock/v3/src/migration/versions/v240/configs"
	"github.com/go-xmlfmt/xmlfmt"
	"os"
	"strconv"
	"strings"
)

//go:embed v240/migration_v240_config_map.xml
var migrationV240ConfigMapXML []byte

func V240() {
	execPath := paths.GetExecDirPath() + "/projects/"
	execProjectsDirs := paths.GetDirs(execPath)

	mapping, err := config2.GetXmlMapFromBytes(migrationV240ConfigMapXML)

	if err != nil {
		logger.Fatalln(err)
	}
	mappingData := config2.ComposeConfigMap(mapping["default"].(map[string]interface{}))

	if paths.IsFileExist(execPath + "config.txt") {
		configData := configs.GetProjectsGeneralConfig()

		resultData := make(map[string]interface{})
		for key, value := range mappingData {
			if v, ok := configData[value]; ok {
				resultData["scopes/default/"+key] = v
			}
		}

		resultMapData := config2.SetXmlMap(resultData)
		w := &bytes.Buffer{}
		w.WriteString(xml.Header)
		err = config2.MarshalXML(resultMapData, xml.NewEncoder(w), "config")
		if err != nil {
			logger.Fatalln(err)
		}

		err = os.WriteFile(execPath+"config.xml", []byte(xmlfmt.FormatXML(w.String(), "", "    ", true)), 0755)
		if err != nil {
			logger.Fatalln(err)
		}
	}

	for _, projectName := range execProjectsDirs {
		if paths.IsFileExist(execPath + projectName + "/env.txt") {
			configData := configs.GetProjectConfigOnly(projectName)
			resultData := make(map[string]interface{})
			for key, value := range mappingData {
				if v, ok := configData[value]; ok {
					resultData["scopes/default/"+key] = v
				}
			}

			if v, ok := configData["HOSTS"]; ok {
				hosts := strings.Split(v, " ")
				runCode := ""
				for key, host := range hosts {
					splitHost := strings.Split(host, ":")
					runCode = "base"
					if key > 0 {
						runCode += strconv.Itoa(key + 1)
					}
					if len(splitHost) > 1 {
						runCode = splitHost[1]
					}
					resultData["scopes/default/nginx/hosts/"+runCode+"/name"] = splitHost[0]
				}
			}

			resultMapData := config2.SetXmlMap(resultData)
			w := &bytes.Buffer{}
			w.WriteString(xml.Header)
			err = config2.MarshalXML(resultMapData, xml.NewEncoder(w), "config")
			if err != nil {
				logger.Fatalln(err)
			}

			err = os.WriteFile(execPath+projectName+"/config.xml", []byte(xmlfmt.FormatXML(w.String(), "", "    ", true)), 0755)
			if err != nil {
				logger.Fatalln(err)
			}
		}

		fixExtendedFiles(mappingData)
	}

	//fixSrcFiles(mappingData)
	//fixDockerFiles(mappingData)
	//fixScriptsFiles(mappingData)
}

func fixExtendedFiles(mapNames map[string]string) {
	projectsPath := paths.GetExecDirPath() + "/projects"
	dirs := paths.GetDirs(projectsPath)
	for _, val := range dirs {
		dockerFiles := paths.GetFilesRecursively(projectsPath + "/" + val + "/docker")
		if len(dockerFiles) > 0 {
			for _, pth := range dockerFiles {
				b, err := os.ReadFile(pth)
				if err == nil {
					str := string(b)
					for to, from := range mapNames {
						str = strings.Replace(str, "\""+from+"\"", "\""+to+"\"", -1)
						str = strings.Replace(str, "{{{"+from+"}}}", "{{{"+to+"}}}", -1)
					}
					str = strings.Replace(str, " ubuntu:{{{", " {{{os/name}}}:{{{", -1)
					str = strings.Replace(str, "{{{NGINX_PROJECT_PORT+", "{{{nginx/port/project+", -1)
					str = strings.Replace(str, "{{{NGINX_PORT+", "{{{nginx/port/default+", -1)
					str = strings.Replace(str, "{{{HOSTS}}}", "{{{hosts}}}", -1)
					str = strings.Replace(str, "\"HOSTS\"", "\"hosts\"", -1)

					err = os.WriteFile(pth, []byte(str), 0755)
					if err != nil {
						logger.Fatalln(err)
					}
				}
			}
		}
	}
}

func fixSrcFiles(mapNames map[string]string) {
	projectsPath := paths.GetExecDirPath() + "/src"
	dockerFiles := paths.GetFilesRecursively(projectsPath)
	if len(dockerFiles) > 0 {
		for _, pth := range dockerFiles {
			if !strings.Contains(pth, "migration") {
				b, err := os.ReadFile(pth)
				if err == nil {
					str := string(b)
					for to, from := range mapNames {
						str = strings.Replace(str, "\""+from+"\"", "\""+to+"\"", -1)
						str = strings.Replace(str, "{{{"+from+"}}}", "{{{"+to+"}}}", -1)
					}
					str = strings.Replace(str, "{{{NGINX_PROJECT_PORT+", "{{{nginx/port/project+", -1)
					str = strings.Replace(str, "{{{NGINX_PORT+", "{{{nginx/port/default+", -1)
					str = strings.Replace(str, "{{{HOSTS}}}", "{{{hosts}}}", -1)
					str = strings.Replace(str, "\"HOSTS\"", "\"hosts\"", -1)

					err = os.WriteFile(pth, []byte(str), 0755)
					if err != nil {
						logger.Fatalln(err)
					}
				}
			}
		}
	}
}

func fixScriptsFiles(mapNames map[string]string) {
	projectsPath := paths.GetExecDirPath() + "/scripts"
	dockerFiles := paths.GetFilesRecursively(projectsPath)
	if len(dockerFiles) > 0 {
		for _, pth := range dockerFiles {
			b, err := os.ReadFile(pth)
			if err == nil {
				str := string(b)
				for to, from := range mapNames {
					str = strings.Replace(str, "\""+from+"\"", "\""+to+"\"", -1)
					str = strings.Replace(str, "{{{"+from+"}}}", "{{{"+to+"}}}", -1)
				}
				str = strings.Replace(str, "{{{HOSTS}}}", "{{{hosts}}}", -1)
				str = strings.Replace(str, "\"HOSTS\"", "\"hosts\"", -1)

				err = os.WriteFile(pth, []byte(str), 0755)
				if err != nil {
					logger.Fatalln(err)
				}
			}
		}
	}
}

func fixDockerFiles(mapNames map[string]string) {
	projectsPath := paths.GetExecDirPath() + "/docker"
	dockerFiles := paths.GetFilesRecursively(projectsPath)
	if len(dockerFiles) > 0 {
		for _, pth := range dockerFiles {
			b, err := os.ReadFile(pth)
			if err == nil {
				str := string(b)
				for to, from := range mapNames {
					str = strings.Replace(str, "{{{"+from+"}}}", "{{{"+to+"}}}", -1)
				}
				str = strings.Replace(str, " ubuntu:{{{", " {{{os/name}}}:{{{", -1)
				str = strings.Replace(str, "{{{NGINX_PROJECT_PORT+", "{{{nginx/port/project+", -1)
				str = strings.Replace(str, "{{{HOSTS}}}", "{{{hosts}}}", -1)
				str = strings.Replace(str, "\"HOSTS\"", "\"hosts\"", -1)
				str = strings.Replace(str, "{{{NGINX_PORT+", "{{{nginx/port/default+", -1)

				err = os.WriteFile(pth, []byte(str), 0755)
				if err != nil {
					logger.Fatalln(err)
				}
			}
		}
	}
}
