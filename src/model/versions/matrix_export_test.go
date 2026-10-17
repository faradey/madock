package versions

import (
	"encoding/json"
	"strings"
	"testing"
)

// evaluate answers a platform version from the export alone, the way a reader
// outside madock would: the platform's tools in order, then the common tools it
// does not define, first matching rule wins, <by> reads an earlier answer.
func evaluate(p PlatformMatrix, common CommonMatrix, version string) map[string]string {
	tools := append([]ToolMatrix{}, p.Tools...)
	defined := map[string]bool{}
	for _, t := range p.Tools {
		defined[t.Name] = true
	}
	for _, t := range common.Tools {
		if !defined[t.Name] {
			tools = append(tools, t)
		}
	}
	got := map[string]string{}
	for _, t := range tools {
		input := version
		if t.By != "" {
			input = got[t.By]
			if input == "" {
				continue
			}
		}
		answer, matched := "", false
		for _, r := range t.Rules {
			if (r.Version != "" && strings.TrimSpace(input) == r.Version) || (r.From != "" && AtLeast(input, r.From)) {
				answer, matched = r.Value, true
				break
			}
		}
		if !matched && t.Default != nil {
			answer = *t.Default
		}
		got[t.Name] = answer
	}
	return got
}

// The export is only worth reading if it is the whole matrix: every version a
// rule names, and every preset, must get from the exported rules exactly what
// madock itself answers. Drop a rule, a default or a <by> from the export and
// some version answers differently.
func TestTheExportAnswersWhatTheMatrixAnswers(t *testing.T) {
	all := ExportMatrix()
	if len(all.Platforms) != len(MatrixPlatforms()) {
		t.Fatalf("exported %d platforms, the matrix has %d", len(all.Platforms), len(MatrixPlatforms()))
	}
	checked := 0
	for _, p := range all.Platforms {
		probes := map[string]bool{"": true}
		for _, pr := range p.Presets {
			probes[pr.Version] = true
		}
		for _, tool := range append(append([]ToolMatrix{}, p.Tools...), all.Common.Tools...) {
			for _, r := range tool.Rules {
				probes[r.From] = true
				probes[r.Version] = true
			}
		}
		for version := range probes {
			want, _ := FromMatrix(p.Name, version)
			got := evaluate(p, all.Common, version)
			for name, field := range toolFields {
				if got[name] != *field(&want) {
					t.Errorf("%s %q: %s is %q from the export, %q from the matrix", p.Name, version, name, got[name], *field(&want))
				}
			}
			checked++
		}
	}
	if checked < 100 {
		t.Fatalf("only %d platform versions checked; the probe set is broken", checked)
	}
}

func TestExportedPresetsCarryFilledDescriptions(t *testing.T) {
	p, ok := ExportPlatform("magento2")
	if !ok || len(p.Presets) == 0 {
		t.Fatal("magento2 exports no presets")
	}
	for _, pr := range p.Presets {
		if strings.Contains(pr.Description, "{") {
			t.Errorf("preset %s: description not filled: %q", pr.Version, pr.Description)
		}
	}
}

func TestExportPlatformRefusesCommonAndUnknown(t *testing.T) {
	for _, name := range []string{"common", "nope", ""} {
		if _, ok := ExportPlatform(name); ok {
			t.Errorf("ExportPlatform(%q) answered", name)
		}
	}
}

// A null default and an empty one mean different things — no answer at all
// against an answer of nothing — so the JSON keeps them apart.
func TestDefaultIsNullOnlyWhenAbsent(t *testing.T) {
	data, err := json.Marshal(ExportMatrix())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"default":null`) {
		t.Error("no tool exports a null default")
	}
	if !strings.Contains(string(data), `"default":"NotCompatible"`) {
		t.Error("magento2's OpenSearch NotCompatible default is missing")
	}
}
