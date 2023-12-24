package versions

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/go-xmlfmt/xmlfmt"
	"github.com/sbabiv/xml2map"
	"log"
	"os"
	"strings"
)

func V240() {
	//execProjectsDirs := paths.GetDirs(paths.GetExecDirPath() + "/projects")
	execPath := paths.GetExecDirPath() + "/projects/"
	//projectName := ""
	if paths.IsFileExist(execPath + "config.xml") {
		err := os.Rename(execPath+"config.xml", execPath+"config.xml.old")
		if err != nil {
			return
		}
	}

	if paths.IsFileExist(execPath + "config.txt") {
		mapping, err := getXmlMap(paths.GetExecDirPath() + "/src/migration/versions/v240/migration_v240_config_map.xml")

		if err != nil {
			log.Fatalln(err)
		}

		mappingData := composeConfigMap(mapping["default"].(map[string]interface{}))

		configData := configs.GetProjectsGeneralConfig()

		resultData := make(map[string]interface{})
		for key, value := range mappingData {
			if v, ok := configData[value]; ok {
				resultData[key] = v
			}
		}
		resultMapData := setXmlMap(resultData)
		w := &bytes.Buffer{}
		w.WriteString(xml.Header)
		err = MarshalXML(resultMapData, xml.NewEncoder(w), xml.StartElement{Name: xml.Name{Local: "default"}})
		if err != nil {
			log.Fatalln(err)
		}

		err = os.WriteFile(execPath+"config.xml", []byte(xmlfmt.FormatXML(w.String(), "", "    ", true)), 0755)
		if err != nil {
			log.Fatalln(err)
		}
	}
	/*envFile := ""
	for _, dir := range execProjectsDirs {
		if paths.IsFileExist(execPath + dir + "/env.txt") {
			if paths.IsFileExist(execPath + dir + "/config.xml") {
				os.Rename(execPath+dir+"/config.xml", execPath+dir+"/config.xml.old")
			}
			projectName = dir
			projectConfOnly := configs.GetProjectConfigOnly(projectName)
			projectConf := configs.GetProjectConfig(projectName)
			envFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
		}
	}*/

	log.Fatalln("Migration v240 is not implemented yet")
}

func getXmlMap(path string) (map[string]interface{}, error) {
	dataByte, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data := string(dataByte)
	decoder := xml2map.NewDecoder(strings.NewReader(data))
	result, err := decoder.Decode()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func setXmlMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range data {
		keys := strings.Split(key, "/")
		switch len(keys) {
		case 1:
			result[keys[0]] = value.(string)
		case 2:
			if _, ok := result[keys[0]]; !ok {
				result[keys[0]] = make(map[string]interface{})
			}
			m, _ := result[keys[0]].(map[string]interface{})
			m[keys[1]] = value.(string)
			result[keys[0]] = m
		case 3:
			if _, ok := result[keys[0]]; !ok {
				result[keys[0]] = make(map[string]interface{})
			}
			if _, ok := result[keys[0]].(map[string]interface{})[keys[1]]; !ok {
				result[keys[0]].(map[string]interface{})[keys[1]] = make(map[string]interface{})
			}
			m := result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})
			m[keys[2]] = value.(string)
			result[keys[0]] = m
		case 4:
			if _, ok := result[keys[0]]; !ok {
				result[keys[0]] = make(map[string]interface{})
			}
			if _, ok := result[keys[0]].(map[string]interface{})[keys[1]]; !ok {
				result[keys[0]].(map[string]interface{})[keys[1]] = make(map[string]interface{})
			}
			if _, ok := result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]]; !ok {
				result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]] = make(map[string]interface{})
			}
			m := result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})
			m[keys[3]] = value.(string)
			result[keys[0]] = m
		case 5:
			if _, ok := result[keys[0]]; !ok {
				result[keys[0]] = make(map[string]interface{})
			}
			if _, ok := result[keys[0]].(map[string]interface{})[keys[1]]; !ok {
				result[keys[0]].(map[string]interface{})[keys[1]] = make(map[string]interface{})
			}
			if _, ok := result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]]; !ok {
				result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]] = make(map[string]interface{})
			}
			if _, ok := result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})[keys[3]]; !ok {
				result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})[keys[3]] = make(map[string]interface{})
			}
			m := result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})[keys[3]].(map[string]interface{})
			m[keys[4]] = value.(string)
			result[keys[0]] = m
		}
	}

	return result
}

func composeConfigMap(rawData map[string]interface{}) map[string]string {
	data := make(map[string]string)
	tempData := make(map[string]string)
	for key, value := range rawData {
		switch value.(type) {
		case string:
			data[key] = value.(string)
		case map[string]interface{}:
			tempData = composeConfigMap(value.(map[string]interface{}))
			for k, v := range tempData {
				data[key+"/"+k] = v

			}
		case []map[string]interface{}:
			for arrKey, arrVal := range value.([]map[string]interface{}) {
				tempData = composeConfigMap(arrVal)
				for k, v := range tempData {
					arrKeyStr := fmt.Sprintf("%d", arrKey)
					data[key+"/"+arrKeyStr+"/"+k] = v
				}
			}
		}
	}

	return data
}

func MarshalXML(s map[string]interface{}, e *xml.Encoder, start xml.StartElement) error {
	var err error
	tokens := []xml.Token{start}
	tokens, err = getXMLTokens(s, e, tokens)
	if err != nil {
		return err
	}
	tokens = append(tokens, xml.EndElement{Name: start.Name})

	for _, t := range tokens {
		err = e.EncodeToken(t)
		if err != nil {
			return err
		}
	}

	// flush to ensure tokens are written
	err = e.Flush()
	if err != nil {
		return err
	}

	return nil
}

func getXMLTokens(s map[string]interface{}, e *xml.Encoder, tokens []xml.Token) ([]xml.Token, error) {
	var err error
	for key, value := range s {
		t := xml.StartElement{Name: xml.Name{Local: key}}
		tokens = append(tokens, t)
		switch value.(type) {
		case string:
			tokens = append(tokens, xml.CharData(value.(string)))
		case map[string]interface{}:
			tokens, err = getXMLTokens(value.(map[string]interface{}), e, tokens)
			if err != nil {
				return nil, err
			}
		}
		tokens = append(tokens, xml.EndElement{Name: t.Name})
	}

	return tokens, nil
}
