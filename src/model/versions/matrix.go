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
// The files are shaped like config.xml: elements only, no attributes.
//
//	<config>
//	    <name>magento2</name>
//	    <tools>
//	        <php>
//	            <rule><from>2.4.9</from><value>8.5</value></rule>
//	            <rule><from>2.4.8</from><value>8.4</value></rule>
//	        </php>
//	        <opensearch>
//	            <rule><version>2.3.7-p4</version><value>1.2.0</value></rule>
//	            <default>NotCompatible</default>
//	        </opensearch>
//	        <xdebug><by>php</by> … </xdebug>
//	    </tools>
//	    <presets>
//	        <preset>
//	            <version>2.4.9</version>
//	            <name>Magento 2.4.9 (Latest)</name>
//	            <description>PHP {php}, OpenSearch {opensearch}</description>
//	        </preset>
//	    </presets>
//	    <budgets>
//	        <search><opensearch><heap>2g</heap></opensearch></search>
//	    </budgets>
//	</config>
//
// Rules are read top to bottom and the first that matches wins: <from> is
// "this version or later" (compared as versions — see AtLeast), <version> is
// exactly that string, which is how named variants such as Shopify's
// "hydrogen" are written. No match gives <default>, or nothing. <by> asks a
// tool about another tool's answer instead of the platform version — Xdebug
// follows PHP — and a tool whose input is empty answers nothing.
//
// matrix/common.xml holds tools every platform shares unless its own file
// defines them, and the lists setup's menus offer.

//go:embed matrix/*.xml
var matrixFS embed.FS

// The model the rest of the package works with.

type matrixFile struct {
	Name    string
	Tools   []matrixTool
	Presets []PresetSpec
	Offers  []matrixOffer
	Budgets map[string]string
}

type matrixTool struct {
	Name    string
	By      string
	Rules   []matrixRule
	Default *string
}

type matrixRule struct {
	From    string
	Version string
	Value   string
}

// matrixOffer is what setup's menu lists for one service, newest first. The
// lists used to be literals in helper/setup/tools, a third copy beside the
// matrix and the presets, and the matrix's own default was sometimes not on
// them — OpenSearch 2.5.0, the default for Magento 2.4.6, could be reached only
// through "Custom". The menu now also always shows the default it was handed;
// see tools.SelectInteractive.
type matrixOffer struct {
	Name     string
	Versions []string
}

// PresetSpec is a named setup choice: a platform version the matrix is asked
// about, with a name and a description for the menu. The description may name
// tools in braces — "PHP {php}" — so it says what the matrix gives and cannot
// fall behind it.
type PresetSpec struct {
	Version     string `xml:"version"`
	Name        string `xml:"name"`
	Description string `xml:"description"`
}

// BudgetKeys are the config keys a platform may set in <budgets>: what a
// program inside a container takes for itself, not what the container may use.
// The database's memory split into its buffer pool and caches, Java's heap for
// Elasticsearch and OpenSearch, Memcached's cache size — and, because the JVM
// lives beyond its heap and cannot run without one, the search engines'
// container limit. Every other container limit is madock Pro's resources:*.
var BudgetKeys = map[string]bool{
	"db/memory":                         true,
	"search/elasticsearch/memory_limit": true,
	"search/elasticsearch/heap":         true,
	"search/opensearch/memory_limit":    true,
	"search/opensearch/heap":            true,
	"memcached/memory":                  true,
}

// The shape of the files.

type rawFile struct {
	Name    string       `xml:"name"`
	Tools   rawTools     `xml:"tools"`
	Offers  rawOffers    `xml:"offers"`
	Presets []PresetSpec `xml:"presets>preset"`
	Budgets rawNode      `xml:"budgets"`
}

type rawTools struct {
	Items []rawTool `xml:",any"`
}

type rawTool struct {
	XMLName xml.Name
	By      string    `xml:"by"`
	Rules   []rawRule `xml:"rule"`
	Default *string   `xml:"default"`
}

type rawRule struct {
	From    *string `xml:"from"`
	Version *string `xml:"version"`
	Value   *string `xml:"value"`
}

type rawOffers struct {
	Items []rawOffer `xml:",any"`
}

type rawOffer struct {
	XMLName  xml.Name
	Versions []string `xml:"v"`
}

type rawNode struct {
	XMLName  xml.Name
	Value    string    `xml:",chardata"`
	Children []rawNode `xml:",any"`
}

// flatten turns nested elements into config keys, the way config.xml is read:
// <search><opensearch><heap>2g</heap></opensearch></search> is
// search/opensearch/heap = 2g.
func flatten(n rawNode, prefix string, out map[string]string) {
	for _, c := range n.Children {
		key := c.XMLName.Local
		if prefix != "" {
			key = prefix + "/" + key
		}
		if len(c.Children) == 0 {
			out[key] = strings.TrimSpace(c.Value)
			continue
		}
		flatten(c, key, out)
	}
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
		f, err := parseMatrix(data)
		if err != nil {
			return nil, fmt.Errorf("matrix/%s: %w", e.Name(), err)
		}
		out[f.Name] = f
	}
	return out, nil
}

func parseMatrix(data []byte) (matrixFile, error) {
	var raw rawFile
	if err := xml.Unmarshal(data, &raw); err != nil {
		return matrixFile{}, err
	}
	f := matrixFile{Name: strings.TrimSpace(raw.Name), Presets: raw.Presets, Budgets: map[string]string{}}
	if f.Name == "" {
		return matrixFile{}, fmt.Errorf("<name> is missing")
	}

	seen := map[string]bool{}
	for _, rt := range raw.Tools.Items {
		name := rt.XMLName.Local
		if _, ok := toolFields[name]; !ok {
			return matrixFile{}, fmt.Errorf("unknown tool <%s>", name)
		}
		if seen[name] {
			return matrixFile{}, fmt.Errorf("tool <%s> is defined twice", name)
		}
		seen[name] = true
		by := strings.TrimSpace(rt.By)
		if by != "" {
			if _, ok := toolFields[by]; !ok {
				return matrixFile{}, fmt.Errorf("tool <%s> follows unknown tool <%s>", name, by)
			}
		}
		tool := matrixTool{Name: name, By: by, Default: rt.Default}
		for _, rr := range rt.Rules {
			if (rr.From == nil) == (rr.Version == nil) {
				return matrixFile{}, fmt.Errorf("a <rule> of <%s> needs exactly one of <from> or <version>", name)
			}
			if rr.Value == nil {
				return matrixFile{}, fmt.Errorf("a <rule> of <%s> has no <value> — write <value/> for none", name)
			}
			r := matrixRule{Value: *rr.Value}
			if rr.From != nil {
				r.From = strings.TrimSpace(*rr.From)
			} else {
				r.Version = strings.TrimSpace(*rr.Version)
			}
			tool.Rules = append(tool.Rules, r)
		}
		f.Tools = append(f.Tools, tool)
	}

	offered := map[string]bool{}
	for _, ro := range raw.Offers.Items {
		name := ro.XMLName.Local
		if len(ro.Versions) == 0 {
			return matrixFile{}, fmt.Errorf("offer <%s> needs at least one <v>", name)
		}
		if offered[name] {
			return matrixFile{}, fmt.Errorf("offer <%s> is defined twice", name)
		}
		offered[name] = true
		f.Offers = append(f.Offers, matrixOffer{Name: name, Versions: ro.Versions})
	}

	flatten(raw.Budgets, "", f.Budgets)
	for key, value := range f.Budgets {
		if !BudgetKeys[key] {
			return matrixFile{}, fmt.Errorf("%s is not a budget a platform may set", key)
		}
		if value == "" {
			return matrixFile{}, fmt.Errorf("budget %s has no value", key)
		}
	}
	return f, nil
}

// PlatformBudgets is a platform's own budgets, by config key.
func PlatformBudgets(platform string) map[string]string {
	out := map[string]string{}
	f, ok := MustLoadMatrices()[platform]
	if !ok {
		return out
	}
	for k, v := range f.Budgets {
		out[k] = v
	}
	return out
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
// does not define; a <by> tool reads the answer of a tool before it.
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

// AnswerFor asks one tool of one file about an input directly — for a <by>
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
	for i, p := range f.Presets {
		out[i] = PresetSpec{
			Version:     strings.TrimSpace(p.Version),
			Name:        strings.TrimSpace(p.Name),
			Description: strings.TrimSpace(p.Description),
		}
	}
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
