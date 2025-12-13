package configs

import (
	"bytes"
	"encoding/xml"
	"os"
	"strings"
	"testing"
)

func TestComposeConfigMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]string
	}{
		{
			name: "simple string values",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "nested map",
			input: map[string]interface{}{
				"parent": map[string]interface{}{
					"child": "value",
				},
			},
			expected: map[string]string{
				"parent/child": "value",
			},
		},
		{
			name: "deeply nested map",
			input: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": "deep_value",
					},
				},
			},
			expected: map[string]string{
				"level1/level2/level3": "deep_value",
			},
		},
		{
			name: "mixed values",
			input: map[string]interface{}{
				"simple": "value",
				"nested": map[string]interface{}{
					"inner": "inner_value",
				},
			},
			expected: map[string]string{
				"simple":       "value",
				"nested/inner": "inner_value",
			},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComposeConfigMap(tt.input)

			if len(got) != len(tt.expected) {
				t.Errorf("ComposeConfigMap() returned %d items, want %d", len(got), len(tt.expected))
			}

			for key, expectedVal := range tt.expected {
				if gotVal, ok := got[key]; !ok {
					t.Errorf("ComposeConfigMap() missing key %q", key)
				} else if gotVal != expectedVal {
					t.Errorf("ComposeConfigMap()[%q] = %q, want %q", key, gotVal, expectedVal)
				}
			}
		})
	}
}

func TestSetXmlMap(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]interface{}
		checkKey    string
		expectExist bool
	}{
		{
			name: "activeScope key",
			input: map[string]interface{}{
				"activeScope": "default",
			},
			checkKey:    "activeScope",
			expectExist: true,
		},
		{
			name: "3-level key",
			input: map[string]interface{}{
				"scopes/default/name": "default",
			},
			checkKey:    "scopes",
			expectExist: true,
		},
		{
			name: "4-level key",
			input: map[string]interface{}{
				"scopes/default/php/version": "8.2",
			},
			checkKey:    "scopes",
			expectExist: true,
		},
		{
			name: "2-level key ignored",
			input: map[string]interface{}{
				"some/key": "value",
			},
			checkKey:    "some",
			expectExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SetXmlMap(tt.input)
			_, exists := got[tt.checkKey]
			if exists != tt.expectExist {
				t.Errorf("SetXmlMap() key %q exists=%v, want %v", tt.checkKey, exists, tt.expectExist)
			}
		})
	}
}

func TestMarshalXML(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		startTag string
		contains []string
	}{
		{
			name: "simple values",
			input: map[string]interface{}{
				"name":    "test",
				"version": "1.0",
			},
			startTag: "config",
			contains: []string{"<config>", "</config>", "<name>test</name>", "<version>1.0</version>"},
		},
		{
			name: "nested values",
			input: map[string]interface{}{
				"php": map[string]interface{}{
					"version": "8.2",
				},
			},
			startTag: "config",
			contains: []string{"<config>", "</config>", "<php>", "</php>", "<version>8.2</version>"},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			startTag: "root",
			contains: []string{"<root>", "</root>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			encoder := xml.NewEncoder(&buf)
			encoder.Indent("", "  ")

			err := MarshalXML(tt.input, encoder, tt.startTag)
			if err != nil {
				t.Fatalf("MarshalXML() error = %v", err)
			}

			result := buf.String()
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("MarshalXML() result missing %q\nGot: %s", substr, result)
				}
			}
		})
	}
}

func TestMarshalXMLDeterministicOrder(t *testing.T) {
	// Test that XML output is deterministic (keys are sorted)
	input := map[string]interface{}{
		"zebra":    "z",
		"apple":    "a",
		"mango":    "m",
		"banana":   "b",
		"cherry":   "c",
		"date":     "d",
		"elephant": "e",
	}

	// Run multiple times and check output is the same
	var firstResult string
	for i := 0; i < 10; i++ {
		var buf bytes.Buffer
		encoder := xml.NewEncoder(&buf)
		err := MarshalXML(input, encoder, "config")
		if err != nil {
			t.Fatalf("MarshalXML() error = %v", err)
		}

		result := buf.String()
		if i == 0 {
			firstResult = result
		} else if result != firstResult {
			t.Errorf("MarshalXML() not deterministic:\nFirst: %s\nGot: %s", firstResult, result)
		}
	}

	// Check that keys appear in alphabetical order
	if !strings.Contains(firstResult, "<apple>") {
		t.Error("Missing <apple> tag")
	}

	// Verify alphabetical ordering
	appleIdx := strings.Index(firstResult, "<apple>")
	bananaIdx := strings.Index(firstResult, "<banana>")
	zebraIdx := strings.Index(firstResult, "<zebra>")

	if appleIdx > bananaIdx || bananaIdx > zebraIdx {
		t.Errorf("Tags not in alphabetical order: apple=%d, banana=%d, zebra=%d", appleIdx, bananaIdx, zebraIdx)
	}
}

func TestGetXmlMap(t *testing.T) {
	// Create a temp XML file
	tmpFile, err := os.CreateTemp("", "testxml*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <name>test</name>
    <version>1.0</version>
    <nested>
        <value>nested_value</value>
    </nested>
</config>`

	tmpFile.WriteString(xmlContent)
	tmpFile.Close()

	result, err := GetXmlMap(tmpPath)
	if err != nil {
		t.Fatalf("GetXmlMap() error = %v", err)
	}

	if _, ok := result["config"]; !ok {
		t.Error("GetXmlMap() missing 'config' key")
	}
}

func TestGetXmlMapNonExistentFile(t *testing.T) {
	_, err := GetXmlMap("/nonexistent/path/config.xml")
	if err == nil {
		t.Error("GetXmlMap() should return error for non-existent file")
	}
}

func TestGetXmlMapInvalidXml(t *testing.T) {
	// Create a temp file with invalid XML
	tmpFile, err := os.CreateTemp("", "invalidxml*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	tmpFile.WriteString("this is not valid xml <unclosed>")
	tmpFile.Close()

	_, err = GetXmlMap(tmpPath)
	if err == nil {
		t.Error("GetXmlMap() should return error for invalid XML")
	}
}

func TestParseXmlFile(t *testing.T) {
	// Create a temp XML file
	tmpFile, err := os.CreateTemp("", "parsexml*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <name>test_project</name>
    <php>
        <version>8.2</version>
    </php>
    <db>
        <host>localhost</host>
        <port>3306</port>
    </db>
</config>`

	tmpFile.WriteString(xmlContent)
	tmpFile.Close()

	result := ParseXmlFile(tmpPath)

	expected := map[string]string{
		"name":        "test_project",
		"php/version": "8.2",
		"db/host":     "localhost",
		"db/port":     "3306",
	}

	for key, expectedVal := range expected {
		if gotVal, ok := result[key]; !ok {
			t.Errorf("ParseXmlFile() missing key %q", key)
		} else if gotVal != expectedVal {
			t.Errorf("ParseXmlFile()[%q] = %q, want %q", key, gotVal, expectedVal)
		}
	}
}

func TestParseFile(t *testing.T) {
	// Create a temp file with key=value pairs
	tmpFile, err := os.CreateTemp("", "parsefile*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	content := `# This is a comment
KEY1=value1
KEY2=value2
EMPTY_KEY=
# Another comment
KEY3=value with spaces`

	tmpFile.WriteString(content)
	tmpFile.Close()

	result := ParseFile(tmpPath)

	expected := map[string]string{
		"KEY1":      "value1",
		"KEY2":      "value2",
		"EMPTY_KEY": "",
		"KEY3":      "value with spaces",
	}

	if len(result) != len(expected) {
		t.Errorf("ParseFile() returned %d items, want %d", len(result), len(expected))
	}

	for key, expectedVal := range expected {
		if gotVal, ok := result[key]; !ok {
			t.Errorf("ParseFile() missing key %q", key)
		} else if gotVal != expectedVal {
			t.Errorf("ParseFile()[%q] = %q, want %q", key, gotVal, expectedVal)
		}
	}
}

func TestMarshalXMLNestedStartTag(t *testing.T) {
	input := map[string]interface{}{
		"name": "value",
	}

	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)

	err := MarshalXML(input, encoder, "root/nested")
	if err != nil {
		t.Fatalf("MarshalXML() error = %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "<root>") {
		t.Error("MarshalXML() missing <root> tag")
	}
	if !strings.Contains(result, "<nested>") {
		t.Error("MarshalXML() missing <nested> tag")
	}
	if !strings.Contains(result, "</root>") {
		t.Error("MarshalXML() missing </root> tag")
	}
	if !strings.Contains(result, "</nested>") {
		t.Error("MarshalXML() missing </nested> tag")
	}
}
