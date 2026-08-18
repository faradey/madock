package configs

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/sbabiv/xml2map"
	"os"
	"sort"
	"strconv"
	"strings"
)

func ParseXmlFile(path string) (conf map[string]string) {
	mapping, err := GetXmlMap(path)

	if err != nil {
		logger.Fatalln(err)
	}

	mappingData := make(map[string]string)
	if _, ok := mapping["config"]; ok {
		mappingData = ComposeConfigMap(mapping["config"].(map[string]interface{}))
	}

	if conf == nil {
		conf = make(map[string]string)
	}

	for key, value := range mappingData {
		conf[key] = decryptIfSecret(key, value)
	}

	return conf
}

func ParseXmlBytes(data []byte) (conf map[string]string) {
	mapping, err := GetXmlMapFromBytes(data)

	if err != nil {
		logger.Fatalln(err)
	}

	mappingData := make(map[string]string)
	if _, ok := mapping["config"]; ok {
		mappingData = ComposeConfigMap(mapping["config"].(map[string]interface{}))
	}

	if conf == nil {
		conf = make(map[string]string)
	}

	for key, value := range mappingData {
		conf[key] = decryptIfSecret(key, value)
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
		logger.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		if len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
			rows = append(rows, line)
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Fatal(err)
	}

	return rows
}

func GetXmlMap(path string) (map[string]interface{}, error) {
	dataByte, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return GetXmlMapFromBytes(dataByte)
}

func GetXmlMapFromBytes(dataByte []byte) (map[string]interface{}, error) {
	data := string(dataByte)
	result := make(map[string]interface{})
	if data != "" {
		decoder := xml2map.NewDecoder(strings.NewReader(data))
		var err error
		result, err = decoder.Decode()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// SetXmlMap converts a flat map with "/" separated keys to a nested map structure.
// Example: {"php/version": "8.1"} -> {"php": {"version": "8.1"}}
func SetXmlMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range data {
		keys := strings.Split(key, "/")
		setNestedValue(result, keys, value)
	}
	collapseIndexedMaps(result)
	return result
}

// collapseIndexedMaps turns {"job": {"0": "a", "1": "b"}} back into
// {"job": []string{"a", "b"}} so the writer can emit the repeated tag it was
// parsed from.
//
// A repeated text element — <job>a</job><job>b</job> — has no other way to
// survive a flat string map, so ComposeConfigMap spells it as job/0 and job/1.
// Writing that back verbatim produces <job><0>a</0></job>, and `0` is not a
// legal XML name: the next read of that file fails with "invalid XML name: 0"
// and, since ParseXmlFile is fatal, takes the whole command with it. The
// heuristic is safe in the other direction for the same reason — a config
// cannot legitimately contain a child named `0`, because it could never have
// been parsed.
func collapseIndexedMaps(m map[string]interface{}) {
	for key, value := range m {
		nested, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		collapseIndexedMaps(nested)
		if list, ok := indexedMapToSlice(nested); ok {
			m[key] = list
		}
	}
}

// indexedMapToSlice reports whether every key is a decimal index covering
// 0..len-1 with a string value, and returns the values in index order.
func indexedMapToSlice(m map[string]interface{}) ([]string, bool) {
	if len(m) == 0 {
		return nil, false
	}
	list := make([]string, len(m))
	seen := make([]bool, len(m))
	for key, value := range m {
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 0 || idx >= len(m) || seen[idx] {
			return nil, false
		}
		str, ok := value.(string)
		if !ok {
			return nil, false
		}
		list[idx] = str
		seen[idx] = true
	}
	return list, true
}

// setNestedValue recursively creates nested maps and sets the value at the deepest level.
func setNestedValue(m map[string]interface{}, keys []string, value interface{}) {
	if len(keys) == 0 {
		return
	}

	if len(keys) == 1 {
		// Leaf node — only set if there is no nested map already at this key.
		// A nested map (branch) takes priority over a leaf value.
		if _, isBranch := m[keys[0]].(map[string]interface{}); isBranch {
			return
		}
		m[keys[0]] = value
		return
	}

	// Intermediate node — create or promote to nested map.
	existing, exists := m[keys[0]]
	if !exists {
		m[keys[0]] = make(map[string]interface{})
	} else if _, ok := existing.(map[string]interface{}); !ok {
		// Current value is a leaf (e.g. empty string from <default></default>),
		// but we need a branch here — promote to map.
		m[keys[0]] = make(map[string]interface{})
	}

	// Recurse into the nested map.
	if nested, ok := m[keys[0]].(map[string]interface{}); ok {
		setNestedValue(nested, keys[1:], value)
	}
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
		case []string:
			// A tag repeated with plain text inside — <job>a</job><job>b</job> —
			// which xml2map hands over as []string. Without this case the type
			// switch matches nothing and the key is dropped without a word, so a
			// <jobs> block with two or more <job> lines in it parsed to no jobs at
			// all while a block with exactly one parsed fine. That is how a live
			// project ran with cron started and an empty crontab.
			for arrKey, arrVal := range value.([]string) {
				data[key+"/"+fmt.Sprintf("%d", arrKey)] = arrVal
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

	// Sort keys to ensure deterministic XML output order
	keys := make([]string, 0, len(s))
	for key := range s {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := s[key]
		t := xml.StartElement{Name: xml.Name{Local: key}}

		// A list is the one shape that is not a single element: it is the same
		// tag written again for each entry, which is how it was read.
		if list, ok := value.([]string); ok {
			for _, item := range list {
				tokens = append(tokens, t, xml.CharData(item), xml.EndElement{Name: t.Name})
			}
			continue
		}

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
