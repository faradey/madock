package versions

import (
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
