package versions

import (
	"regexp"
	"strings"
	"testing"
)

func TestTheFirstMatchingRuleWins(t *testing.T) {
	tool := matrixTool{Rules: []matrixRule{
		{From: "2.4.7", Value: "a"},
		{From: "2.4.6-p11", Value: "b"},
		{From: "2.4.6", Value: "c"},
	}}
	for in, want := range map[string]string{"2.4.8": "a", "2.4.7": "a", "2.4.6-p11": "b", "2.4.6-p4": "c", "2.4.5": ""} {
		if got := tool.answer(in); got != want {
			t.Errorf("answer(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAVersionRuleMatchesExactlyAndCanAnswerNothing(t *testing.T) {
	def := "8.2"
	tool := matrixTool{Rules: []matrixRule{
		{Version: "hydrogen", Value: ""},
		{Version: "api-php", Value: "8.3"},
	}, Default: &def}
	for in, want := range map[string]string{"hydrogen": "", "api-php": "8.3", "": "8.2", "hydrogen2": "8.2"} {
		if got := tool.answer(in); got != want {
			t.Errorf("answer(%q) = %q, want %q", in, got, want)
		}
	}
}

// Xdebug follows PHP, and a stack without PHP gets no Xdebug.
func TestAByToolReadsTheToolItFollows(t *testing.T) {
	if got, _ := FromMatrix("magento2", "2.4.9"); got.Xdebug != "3.5.0" {
		t.Errorf("Magento 2.4.9 (PHP %s): Xdebug %q, want 3.5.0", got.Php, got.Xdebug)
	}
	if got, _ := FromMatrix("medusa", "2"); got.Xdebug != "" {
		t.Errorf("Medusa runs no PHP and should get no Xdebug, got %q", got.Xdebug)
	}
}

func TestEveryEmbeddedFileLoads(t *testing.T) {
	m, err := parseMatrices()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m["common"]; !ok {
		t.Error("common.xml is missing")
	}
	if len(MatrixPlatforms()) < 11 {
		t.Errorf("expected every platform to have a matrix, got %v", MatrixPlatforms())
	}
}

// A preset's description is filled from the versions it gets, so it cannot
// name a version the preset does not install.
func TestAPresetDescriptionSaysWhatThePresetInstalls(t *testing.T) {
	for _, p := range MatrixPlatforms() {
		for _, spec := range Presets(p) {
			v, _ := FromMatrix(p, spec.Version)
			desc := DescribePreset(spec, v)
			if strings.Contains(desc, "{") {
				t.Errorf("%s / %s: placeholder left in %q", p, spec.Name, desc)
			}
		}
	}
}

func TestTheFormatRefusesWhatItCannotRead(t *testing.T) {
	cases := map[string]string{
		"no name":            `<config><tools><php><default>8.2</default></php></tools></config>`,
		"unknown tool":       `<config><name>x</name><tools><phpp><default>8.2</default></phpp></tools></config>`,
		"tool twice":         `<config><name>x</name><tools><php/><php/></tools></config>`,
		"rule without value": `<config><name>x</name><tools><php><rule><from>1</from></rule></php></tools></config>`,
		"rule with both":     `<config><name>x</name><tools><php><rule><from>1</from><version>1</version><value>8</value></rule></php></tools></config>`,
		"rule with neither":  `<config><name>x</name><tools><php><rule><value>8</value></rule></php></tools></config>`,
		"unknown budget":     `<config><name>x</name><budgets><php><memory>1G</memory></php></budgets></config>`,
		"empty budget":       `<config><name>x</name><budgets><db><memory></memory></db></budgets></config>`,
	}
	for name, xmlText := range cases {
		if _, err := parseMatrix([]byte(xmlText)); err == nil {
			t.Errorf("%s: accepted %s", name, xmlText)
		}
	}
}

// Budgets nest the way config.xml does, and are read back as config keys.
func TestBudgetsNestLikeConfigXML(t *testing.T) {
	f, err := parseMatrix([]byte(`<config><name>x</name><budgets>
		<search><opensearch><heap>2g</heap><memory_limit>4g</memory_limit></opensearch></search>
		<db><memory>1G</memory></db>
	</budgets></config>`))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"search/opensearch/heap": "2g", "search/opensearch/memory_limit": "4g", "db/memory": "1G"}
	for k, v := range want {
		if f.Budgets[k] != v {
			t.Errorf("%s = %q, want %q (all: %v)", k, f.Budgets[k], v, f.Budgets)
		}
	}
}

// No attributes anywhere: the files are shaped like config.xml.
func TestNoMatrixFileUsesAttributes(t *testing.T) {
	entries, _ := matrixFS.ReadDir("matrix")
	attr := regexp.MustCompile(`<[a-zA-Z_]+\s+[a-zA-Z_]+=`)
	for _, e := range entries {
		data, _ := matrixFS.ReadFile("matrix/" + e.Name())
		body := regexp.MustCompile(`(?s)<!--.*?-->|<\?xml.*?\?>`).ReplaceAll(data, nil)
		if m := attr.Find(body); m != nil {
			t.Errorf("matrix/%s uses an attribute: %s", e.Name(), m)
		}
	}
}
