package configs

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"github.com/sbabiv/xml2map"
	"log"
	"os"
	"strings"
)

func ParseXmlFile(path string) (conf map[string]string) {
	mapping, err := GetXmlMap(path)

	if err != nil {
		log.Fatalln(err)
	}

	mappingData := make(map[string]string)
	if _, ok := mapping["config"]; ok {
		mappingData = ComposeConfigMap(mapping["config"].(map[string]interface{}))
	}

	if conf == nil {
		conf = make(map[string]string)
	}

	for key, value := range mappingData {
		conf[key] = value
	}

	return conf
}

func ParseFile(path string) (conf map[string]string) {
	conf = make(map[string]string)
	lines := getLines(path)

	for _, line := range lines {
		opt := strings.Split(strings.TrimSpace(line), "=")
		if len(opt) > 1 {
			conf[opt[0]] = opt[1]
		} else if len(opt) > 0 {
			conf[opt[0]] = ""
		}
	}

	return conf
}

func getLines(path string) []string {
	var rows []string
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) > 0 && strings.TrimSpace(line)[:1] != "#" {
			rows = append(rows, line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return rows
}

func GetXmlMap(path string) (map[string]interface{}, error) {
	dataByte, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data := string(dataByte)
	result := make(map[string]interface{})
	if data != "" {
		decoder := xml2map.NewDecoder(strings.NewReader(data))
		result, err = decoder.Decode()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func SetXmlMap(data map[string]interface{}) map[string]interface{} {
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
			result[keys[0]].(map[string]interface{})[keys[1]] = value.(string)
		case 3:
			if _, ok := result[keys[0]]; !ok {
				result[keys[0]] = make(map[string]interface{})
			}
			if _, ok := result[keys[0]].(map[string]interface{})[keys[1]]; !ok {
				result[keys[0]].(map[string]interface{})[keys[1]] = make(map[string]interface{})
			}
			result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]] = value.(string)
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
			result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})[keys[3]] = value.(string)
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
			result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})[keys[3]].(map[string]interface{})[keys[4]] = value.(string)
		case 6:
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
			if _, ok := result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})[keys[3]].(map[string]interface{})[keys[4]]; !ok {
				result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})[keys[3]].(map[string]interface{})[keys[4]] = make(map[string]interface{})
			}
			result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})[keys[3]].(map[string]interface{})[keys[4]].(map[string]interface{})[keys[5]] = value.(string)
		case 7:
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
			if _, ok := result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})[keys[3]].(map[string]interface{})[keys[4]]; !ok {
				result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})[keys[3]].(map[string]interface{})[keys[4]] = make(map[string]interface{})
			}
			if _, ok := result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})[keys[3]].(map[string]interface{})[keys[4]].(map[string]interface{})[keys[5]]; !ok {
				result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})[keys[3]].(map[string]interface{})[keys[4]].(map[string]interface{})[keys[5]] = make(map[string]interface{})
			}
			result[keys[0]].(map[string]interface{})[keys[1]].(map[string]interface{})[keys[2]].(map[string]interface{})[keys[3]].(map[string]interface{})[keys[4]].(map[string]interface{})[keys[5]].(map[string]interface{})[keys[6]] = value.(string)
		}
	}

	return result
}

func ComposeConfigMap(rawData map[string]interface{}) map[string]string {
	data := make(map[string]string)
	tempData := make(map[string]string)
	for key, value := range rawData {
		switch value.(type) {
		case string:
			data[key] = value.(string)
		case map[string]interface{}:
			tempData = ComposeConfigMap(value.(map[string]interface{}))
			for k, v := range tempData {
				data[key+"/"+k] = v
			}
		case []map[string]interface{}:
			for arrKey, arrVal := range value.([]map[string]interface{}) {
				tempData = ComposeConfigMap(arrVal)
				for k, v := range tempData {
					arrKeyStr := fmt.Sprintf("%d", arrKey)
					data[key+"/"+arrKeyStr+"/"+k] = v
				}
			}
		}
	}

	return data
}

func MarshalXML(s map[string]interface{}, e *xml.Encoder, startTag string) error {
	var err error
	var tokens []xml.Token
	var tokensEnd []xml.Token
	startTags := strings.Split(startTag, "/")
	for _, tag := range startTags {
		tokens = append(tokens, xml.StartElement{Name: xml.Name{Local: tag}})
		tokensEnd = append([]xml.Token{xml.EndElement{Name: xml.Name{Local: tag}}}, tokensEnd...)
		if err != nil {
			return err
		}
	}
	tokens, err = getXMLTokens(s, e, tokens)
	if err != nil {
		return err
	}
	tokens = append(tokens, tokensEnd...)

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
