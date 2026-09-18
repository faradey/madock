package info

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"

	inforeg "github.com/faradey/madock/v4/src/info"
)

// A report the way general/info and the Magento handler build one: two
// key/value sections, two scope blocks sharing a key, a table, and a host
// code that repeats — which is the case that makes "#2" keys.
func sampleBlocks() []Block {
	return []Block{
		{Key: "project", Title: "Project", Items: []inforeg.Item{
			{Key: "name", Value: "shop"},
			{Key: "path", Value: "/var/www/shop"},
		}},
		{Key: "hosts", Title: "Hosts", Items: []inforeg.Item{
			{Key: "base", Value: "shop.test"},
			{Key: "base#2", Value: "shop-2.test"},
		}},
		{Key: "scopes", Name: "old", Title: "Scope: old", Items: []inforeg.Item{{Key: "xdebug", Value: "3.2.2"}}},
		{Key: "scopes", Name: "new", Title: "Scope: new", Items: []inforeg.Item{{Key: "xdebug", Value: "3.4.4"}}},
		{Key: "modules", Title: "Third-party modules", RowName: "module",
			Columns: []inforeg.Column{{Key: "name", Title: "Name"}, {Key: "version", Title: "Version"}},
			Rows:    [][]string{{"Vendor_A", "1.0.0"}, {"Vendor_B", "2 < 3 & \"more\""}}},
		{Key: "warnings", Title: "Warning", Lines: []string{"latest versions unknown"}},
	}
}

func TestRenderRefusesUnknownFormat(t *testing.T) {
	var out bytes.Buffer
	err := Render("yaml", sampleBlocks(), &out)
	if err == nil {
		t.Fatal("an unknown format rendered something instead of failing")
	}
	if out.Len() != 0 {
		t.Fatalf("an unknown format wrote %d bytes before failing", out.Len())
	}
	if !strings.Contains(err.Error(), "yaml") {
		t.Errorf("the error does not name the format: %v", err)
	}
}

func TestRenderJSONKeepsOrderAndGroupsScopes(t *testing.T) {
	var out bytes.Buffer
	if err := Render("json", sampleBlocks(), &out); err != nil {
		t.Fatal(err)
	}

	// Order is the point of the ordered encoder: the sections must come out
	// in the order the report lists them, not alphabetically.
	text := out.String()
	for _, pair := range [][2]string{{`"project"`, `"hosts"`}, {`"hosts"`, `"scopes"`}, {`"scopes"`, `"modules"`}, {`"modules"`, `"warnings"`}} {
		if strings.Index(text, pair[0]) > strings.Index(text, pair[1]) {
			t.Errorf("%s printed after %s", pair[0], pair[1])
		}
	}

	var got struct {
		Project map[string]string            `json:"project"`
		Hosts   map[string]string            `json:"hosts"`
		Scopes  map[string]map[string]string `json:"scopes"`
		Modules []map[string]string          `json:"modules"`
		Warns   []string                     `json:"warnings"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, text)
	}
	if got.Hosts["base#2"] != "shop-2.test" {
		t.Errorf("the repeated host code was lost: %v", got.Hosts)
	}
	if got.Scopes["old"]["xdebug"] != "3.2.2" || got.Scopes["new"]["xdebug"] != "3.4.4" {
		t.Errorf("scopes were not grouped under their key: %v", got.Scopes)
	}
	if len(got.Modules) != 2 || got.Modules[1]["version"] != "2 < 3 & \"more\"" {
		t.Errorf("table rows did not survive: %v", got.Modules)
	}
	if len(got.Warns) != 1 {
		t.Errorf("lines block did not survive: %v", got.Warns)
	}
}

func TestRenderXMLIsWellFormedWithAwkwardKeys(t *testing.T) {
	var out bytes.Buffer
	if err := Render("xml", sampleBlocks(), &out); err != nil {
		t.Fatal(err)
	}

	// The parser is the judge: "base#2" as an element name, and "<", "&"
	// and quotes inside a cell, are exactly what an unescaped writer breaks on.
	dec := xml.NewDecoder(bytes.NewReader(out.Bytes()))
	for {
		_, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("not well-formed XML: %v\n%s", err, out.String())
		}
	}

	text := out.String()
	for _, want := range []string{"<base_2>shop-2.test</base_2>", `<scope name="old">`, "<module>", "2 &lt; 3 &amp; &#34;more&#34;", "<line>latest versions unknown</line>"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
}

func TestRenderMarkdownEscapesPipes(t *testing.T) {
	blocks := []Block{{Key: "hosts", Title: "Hosts", Items: []inforeg.Item{{Key: "a|b", Value: "x|y"}}}}
	var out bytes.Buffer
	if err := Render("md", blocks, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `| a\|b | x\|y |`) {
		t.Errorf("a pipe inside a cell breaks the table:\n%s", out.String())
	}
	if !strings.HasPrefix(out.String(), "## Hosts\n") {
		t.Errorf("no heading:\n%s", out.String())
	}
}

func TestRenderTextTableAlignsColumns(t *testing.T) {
	var out bytes.Buffer
	if err := Render("text", sampleBlocks(), &out); err != nil {
		t.Fatal(err)
	}
	// fmtc.Section prints to stdout, not to the writer; the table and the
	// warning lines are what this renderer writes itself.
	text := out.String()
	if !strings.Contains(text, "Vendor_A  1.0.0") {
		t.Errorf("table cells are not padded to the column width:\n%s", text)
	}
	if !strings.Contains(text, "Warning: latest versions unknown") {
		t.Errorf("warning line missing:\n%s", text)
	}
}
