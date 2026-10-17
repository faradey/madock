package versions

import (
	"sort"
	"strings"
)

// The matrix as data for something other than madock. `madock matrix --json`
// prints it, so a tool that builds environments from the same rules — a
// template builder, a UI offering versions — reads every rule at once instead
// of asking FromMatrix one platform version at a time and guessing where the
// boundaries are.
//
// The view is a copy: nothing here can change what the package answers.

// MatrixExport is the whole matrix: every platform file and common.xml.
type MatrixExport struct {
	Platforms []PlatformMatrix `json:"platforms"`
	Common    CommonMatrix     `json:"common"`
}

// PlatformMatrix is one matrix/<platform>.xml.
type PlatformMatrix struct {
	Name string `json:"name"`
	// Tools are in file order, which is the order rules are evaluated in: a
	// <by> tool reads the answer of a tool before it. Common tools the platform
	// does not define are not repeated here; they come after these, from
	// Common.Tools.
	Tools   []ToolMatrix      `json:"tools"`
	Presets []PresetExport    `json:"presets"`
	Budgets map[string]string `json:"budgets"`
}

// CommonMatrix is matrix/common.xml.
type CommonMatrix struct {
	Tools  []ToolMatrix  `json:"tools"`
	Offers []OfferExport `json:"offers"`
}

// ToolMatrix is one tool's rules. The first matching rule wins; none gives
// Default, and a nil Default gives nothing.
type ToolMatrix struct {
	Name    string       `json:"name"`
	By      string       `json:"by,omitempty"`
	Rules   []RuleExport `json:"rules"`
	Default *string      `json:"default"`
}

// RuleExport has exactly one of From ("this version or later", compared as
// versions) and Version (exactly that string).
type RuleExport struct {
	From    string `json:"from,omitempty"`
	Version string `json:"version,omitempty"`
	Value   string `json:"value"`
}

// PresetExport is a preset with its description filled in from what the
// matrix gives its version, as setup's menu shows it.
type PresetExport struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// OfferExport is the list setup's menu offers for one service.
type OfferExport struct {
	Name     string   `json:"name"`
	Versions []string `json:"versions"`
}

// ExportMatrix returns every platform, sorted by name, and common.xml.
func ExportMatrix() MatrixExport {
	out := MatrixExport{Platforms: []PlatformMatrix{}}
	for _, name := range MatrixPlatforms() {
		p, _ := ExportPlatform(name)
		out.Platforms = append(out.Platforms, p)
	}

	common := MustLoadMatrices()["common"]
	out.Common = CommonMatrix{Tools: exportTools(common.Tools), Offers: []OfferExport{}}
	for _, o := range common.Offers {
		out.Common.Offers = append(out.Common.Offers, OfferExport{Name: o.Name, Versions: Offer(o.Name)})
	}
	return out
}

// ExportPlatform returns one platform's file; false when there is none.
// common.xml is not a platform and is not returned here.
func ExportPlatform(platform string) (PlatformMatrix, bool) {
	if platform == "common" {
		return PlatformMatrix{}, false
	}
	f, ok := MustLoadMatrices()[platform]
	if !ok {
		return PlatformMatrix{}, false
	}
	out := PlatformMatrix{
		Name:    f.Name,
		Tools:   exportTools(f.Tools),
		Presets: []PresetExport{},
		Budgets: PlatformBudgets(platform),
	}
	for _, p := range Presets(platform) {
		v, _ := FromMatrix(platform, p.Version)
		out.Presets = append(out.Presets, PresetExport{
			Version:     p.Version,
			Name:        p.Name,
			Description: DescribePreset(p, v),
		})
	}
	return out, true
}

func exportTools(tools []matrixTool) []ToolMatrix {
	out := make([]ToolMatrix, 0, len(tools))
	for _, t := range tools {
		tm := ToolMatrix{Name: t.Name, By: t.By, Rules: make([]RuleExport, 0, len(t.Rules))}
		if t.Default != nil {
			d := strings.TrimSpace(*t.Default)
			tm.Default = &d
		}
		for _, r := range t.Rules {
			tm.Rules = append(tm.Rules, RuleExport{From: r.From, Version: r.Version, Value: strings.TrimSpace(r.Value)})
		}
		out = append(out, tm)
	}
	return out
}

// SortedBudgetKeys is a platform's budget keys in order, for text output.
func SortedBudgetKeys(budgets map[string]string) []string {
	keys := make([]string, 0, len(budgets))
	for k := range budgets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
