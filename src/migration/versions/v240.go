package versions

import (
	"bytes"
	"encoding/xml"
	config2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/migration/versions/v240/configs"
	"github.com/go-xmlfmt/xmlfmt"
	"log"
	"os"
	"strings"
)

func V240() {
	execPath := paths.GetExecDirPath() + "/projects/"
	execProjectsDirs := paths.GetDirs(execPath)
	if paths.IsFileExist(execPath + "config.xml") {
		err := os.Rename(execPath+"config.xml", execPath+"config.xml.old")
		if err != nil {
			log.Fatalln(err)
		}
	}

	mapping, err := config2.GetXmlMap(paths.GetExecDirPath() + "/src/migration/versions/v240/migration_v240_config_map.xml")

	if err != nil {
		log.Fatalln(err)
	}
	mappingData := config2.ComposeConfigMap(mapping["default"].(map[string]interface{}))

	if paths.IsFileExist(execPath + "config.txt") {
		configData := config2.GetProjectsGeneralConfig()

		resultData := make(map[string]interface{})
		for key, value := range mappingData {
			if v, ok := configData[value]; ok {
				resultData[key] = v
			}
		}
		resultMapData := config2.SetXmlMap(resultData)
		w := &bytes.Buffer{}
		w.WriteString(xml.Header)
		err = config2.MarshalXML(resultMapData, xml.NewEncoder(w), "scopes/default")
		if err != nil {
			log.Fatalln(err)
		}

		err = os.WriteFile(execPath+"config.xml", []byte(xmlfmt.FormatXML(w.String(), "", "    ", true)), 0755)
		if err != nil {
			log.Fatalln(err)
		}
	}

	for _, projectName := range execProjectsDirs {
		if paths.IsFileExist(execPath + projectName + "/env.txt") {
			if paths.IsFileExist(execPath + projectName + "/config.xml") {
				err := os.Rename(execPath+projectName+"/config.xml", execPath+projectName+"/config.xml.old")
				if err != nil {
					log.Fatalln(err)
				}
			}

			configData := configs.GetProjectConfigOnly(projectName)
			resultData := make(map[string]interface{})
			for key, value := range mappingData {
				if v, ok := configData[value]; ok {
					resultData[key] = v
				}
			}

			if v, ok := configData["HOSTS"]; ok {
				hosts := strings.Split(v, " ")
				for _, host := range hosts {
					splitHost := strings.Split(host, ":")
					runCode := "base"
					if len(splitHost) > 1 {
						runCode = splitHost[1]
					}
					resultData["nginx/hosts/"+runCode] = splitHost[0]
				}
			}

			resultMapData := config2.SetXmlMap(resultData)
			w := &bytes.Buffer{}
			w.WriteString(xml.Header)
			err = config2.MarshalXML(resultMapData, xml.NewEncoder(w), "scopes/default")
			if err != nil {
				log.Fatalln(err)
			}

			err = os.WriteFile(execPath+projectName+"/config.xml", []byte(xmlfmt.FormatXML(w.String(), "", "    ", true)), 0755)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}

	log.Fatalln("Migration v240 is not implemented yet")
}
