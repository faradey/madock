package versions

import (
	"embed"
	"encoding/xml"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
)

// The compatibility matrix — which tool versions a platform version gets by
// default — lives in matrix/<platform>.xml, one file per platform, compiled
// into the binary. It used to be hand-written Go, a function per tool per
// platform, with a second, hand-kept copy of the same answers in the setup
// presets that had drifted from it in twelve places of twenty-two. Data in
// one place can be read by something other than madock, and the presets are
// now built from it rather than beside it.
//
// A file looks like this:
//
//	<platform name="magento2">
//	  <tool name="php">
//	    <when from="2.4.9">8.5</when>
//	    <when from="2.4.8">8.4</when>
//	  </tool>
//	  <tool name="opensearch">
//	    <when version="2.3.7-p4">1.2.0</when>
//	    <default>NotCompatible</default>
//	  </tool>
//	  <tool name="xdebug" by="php"> … </tool>
//	  <preset version="2.4.9" name="Magento 2.4.9 (Latest)" description="PHP {php}, OpenSearch {opensearch}"/>
//	</platform>
//
// Rules are read top to bottom and the first that matches wins: `from` is
// "this version or later" (compared as versions — see AtLeast), `version` is
// exactly that string, which is how named variants such as Shopify's
// "hydrogen" are written. No match gives <default>, or nothing. `by` asks a
// tool about another tool's answer instead of the platform version — Xdebug
// follows PHP — and a tool whose input is empty answers nothing.
//
// matrix/common.xml holds tools every platform shares unless its own file
// defines them.

//go:embed matrix/*.xml
var matrixFS embed.FS

type matrixFile struct {
	Name    string        `xml:"name,attr"`
	Tools   []matrixTool  `xml:"tool"`
	Presets []PresetSpec  `xml:"preset"`
	Offers  []matrixOffer `xml:"offer"`
}

// matrixOffer is what setup's menu lists for one service, newest first. The
// lists used to be literals in helper/setup/tools, a third copy beside the
// matrix and the presets, and the matrix's own default was sometimes not on
// them — OpenSearch 2.5.0, the default for Magento 2.4.6, could be reached only
// through "Custom". The menu now also always shows the default it was handed;
// see tools.SelectInteractive.
type matrixOffer struct {
	Name     string   `xml:"name,attr"`
	Versions []string `xml:"v"`
}

type matrixTool struct {
	Name    string       `xml:"name,attr"`
	By      string       `xml:"by,attr"`
	Rules   []matrixRule `xml:"when"`
	Default *string      `xml:"default"`
}

type matrixRule struct {
	From    string `xml:"from,attr"`
	Version string `xml:"version,attr"`
	Value   string `xml:",chardata"`
}

// PresetSpec is a named setup choice: a platform version the matrix is asked
// about, with a name and a description for the menu. The description may name
// tools in braces — "PHP {php}" — so it says what the matrix gives and cannot
// fall behind it.
type PresetSpec struct {
	Version     string `xml:"version,attr"`
	Name        string `xml:"name,attr"`
	Description string `xml:"description,attr"`
}

// toolFields maps a tool name in the XML to its ToolsVersions field. A name
// that is not here is refused when the files are loaded, so a typo fails the
// tests instead of quietly answering nothing.
var toolFields = map[string]func(*ToolsVersions) *string{
	"language":     func(t *ToolsVersions) *string { return &t.Language },
	"php":          func(t *ToolsVersions) *string { return &t.Php },
	"dbtype":       func(t *ToolsVersions) *string { return &t.DbType },
	"db":           func(t *ToolsVersions) *string { return &t.Db },
	"searchengine": func(t *ToolsVersions) *string { return &t.SearchEngine },
	"elastic":      func(t *ToolsVersions) *string { return &t.Elastic },
	"opensearch":   func(t *ToolsVersions) *string { return &t.OpenSearch },
	"composer":     func(t *ToolsVersions) *string { return &t.Composer },
	"redis":        func(t *ToolsVersions) *string { return &t.Redis },
	"valkey":       func(t *ToolsVersions) *string { return &t.Valkey },
	"rabbitmq":     func(t *ToolsVersions) *string { return &t.RabbitMQ },
	"xdebug":       func(t *ToolsVersions) *string { return &t.Xdebug },
	"nodejs":       func(t *ToolsVersions) *string { return &t.NodeJs },
	"yarn":         func(t *ToolsVersions) *string { return &t.Yarn },
	"python":       func(t *ToolsVersions) *string { return &t.Python },
	"golang":       func(t *ToolsVersions) *string { return &t.Golang },
	"ruby":         func(t *ToolsVersions) *string { return &t.Ruby },
}

var (
	matrixOnce  sync.Once
	matrices    map[string]matrixFile
	matrixError error
)

func loadMatrices() (map[string]matrixFile, error) {
	matrixOnce.Do(func() {
		matrices, matrixError = parseMatrices()
	})
	return matrices, matrixError
}

func parseMatrices() (map[string]matrixFile, error) {
	entries, err := matrixFS.ReadDir("matrix")
	if err != nil {
		return nil, err
	}
	out := map[string]matrixFile{}
	for _, e := range entries {
		data, err := matrixFS.ReadFile(path.Join("matrix", e.Name()))
		if err != nil {
			return nil, err
		}
		var f matrixFile
		if err := xml.Unmarshal(data, &f); err != nil {
			return nil, fmt.Errorf("matrix/%s: %w", e.Name(), err)
		}
		if f.Name == "" {
			return nil, fmt.Errorf("matrix/%s: <platform> has no name", e.Name())
		}
		seen := map[string]bool{}
		for _, tool := range f.Tools {
			if _, ok := toolFields[tool.Name]; !ok {
				return nil, fmt.Errorf("matrix/%s: unknown tool %q", e.Name(), tool.Name)
			}
			if seen[tool.Name] {
				return nil, fmt.Errorf("matrix/%s: tool %q is defined twice", e.Name(), tool.Name)
			}
			seen[tool.Name] = true
			if tool.By != "" {
				if _, ok := toolFields[tool.By]; !ok {
					return nil, fmt.Errorf("matrix/%s: tool %q is by unknown tool %q", e.Name(), tool.Name, tool.By)
				}
			}
			for _, r := range tool.Rules {
				if (r.From == "") == (r.Version == "") {
					return nil, fmt.Errorf("matrix/%s: a rule of %q needs exactly one of from= or version=", e.Name(), tool.Name)
				}
			}
		}
		offered := map[string]bool{}
		for _, o := range f.Offers {
			if o.Name == "" || len(o.Versions) == 0 {
				return nil, fmt.Errorf("matrix/%s: an <offer> needs a name and at least one <v>", e.Name())
			}
			if offered[o.Name] {
				return nil, fmt.Errorf("matrix/%s: offer %q is defined twice", e.Name(), o.Name)
			}
			offered[o.Name] = true
		}
		out[f.Name] = f
	}
	return out, nil
}

// Offer is what setup's menu lists for a service, from matrix/common.xml.
// An unknown name is a defect in the caller and answers nothing.
func Offer(name string) []string {
	for _, o := range MustLoadMatrices()["common"].Offers {
		if o.Name == name {
			out := make([]string, len(o.Versions))
			for i, v := range o.Versions {
				out[i] = strings.TrimSpace(v)
			}
			return out
		}
	}
	return nil
}

// MustLoadMatrices fails loudly when an embedded file is malformed. The files
// ship inside the binary, so a broken one is a build defect, not a user error.
func MustLoadMatrices() map[string]matrixFile {
	m, err := loadMatrices()
	if err != nil {
		panic(err)
	}
	return m
}

func (tool matrixTool) answer(input string) string {
	for _, r := range tool.Rules {
		if r.Version != "" && strings.TrimSpace(input) == r.Version {
			return strings.TrimSpace(r.Value)
		}
		if r.From != "" && AtLeast(input, r.From) {
			return strings.TrimSpace(r.Value)
		}
	}
	if tool.Default != nil {
		return strings.TrimSpace(*tool.Default)
	}
	return ""
}

// FromMatrix answers what the matrix gives a platform at a version. The
// platform's own tools are evaluated in file order, then the common tools it
// does not define; a `by` tool reads the answer of a tool before it.
func FromMatrix(platform, version string) (ToolsVersions, bool) {
	all := MustLoadMatrices()
	f, ok := all[platform]
	if !ok {
		return ToolsVersions{}, false
	}

	version = strings.TrimSpace(version)
	out := ToolsVersions{Platform: platform, PlatformVersion: version}

	tools := append([]matrixTool{}, f.Tools...)
	defined := map[string]bool{}
	for _, t := range f.Tools {
		defined[t.Name] = true
	}
	for _, t := range all["common"].Tools {
		if !defined[t.Name] {
			tools = append(tools, t)
		}
	}

	for _, tool := range tools {
		input := version
		if tool.By != "" {
			input = *toolFields[tool.By](&out)
			if input == "" {
				continue
			}
		}
		*toolFields[tool.Name](&out) = tool.answer(input)
	}
	return out, true
}

// AnswerFor asks one tool of one file about an input directly — for a `by`
// tool, the value of the tool it follows. GetXdebugVersion is common.xml's
// xdebug asked about a PHP version.
func AnswerFor(platform, tool, input string) string {
	f, ok := MustLoadMatrices()[platform]
	if !ok {
		return ""
	}
	for _, t := range f.Tools {
		if t.Name == tool {
			return t.answer(input)
		}
	}
	return ""
}

// Answer is one tool's value for a platform version.
func Answer(platform, tool, version string) string {
	got, ok := FromMatrix(platform, version)
	if !ok {
		return ""
	}
	field, ok := toolFields[tool]
	if !ok {
		return ""
	}
	return *field(&got)
}

// Presets lists a platform's presets, each with the versions the matrix gives
// its platform version and its description filled in from them.
func Presets(platform string) []PresetSpec {
	f, ok := MustLoadMatrices()[platform]
	if !ok {
		return nil
	}
	out := make([]PresetSpec, len(f.Presets))
	copy(out, f.Presets)
	return out
}

// DescribePreset fills a preset's description from the versions it gets.
func DescribePreset(p PresetSpec, v ToolsVersions) string {
	desc := p.Description
	for name, field := range toolFields {
		desc = strings.ReplaceAll(desc, "{"+name+"}", *field(&v))
	}
	return desc
}

// MatrixPlatforms is every platform with a matrix file, sorted.
func MatrixPlatforms() []string {
	var names []string
	for name := range MustLoadMatrices() {
		if name != "common" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// NotCompatible is the answer a matrix gives for a tool that cannot run with
// that platform version at all — OpenSearch for Magento before 2.4.3-p2.
// Callers check for it; it must never reach a config as a version.
const NotCompatible = "NotCompatible"
