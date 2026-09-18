package info

// Renderers for the report blocks declared in src/info. They live here rather
// than beside the model because src/info is a leaf in .go-arch-lint.yml, and
// colour and section printing are helpers.

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/faradey/madock/v4/src/helper/cli/color"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	inforeg "github.com/faradey/madock/v4/src/info"
)

type Block = inforeg.Block

// Formats lists what Render accepts, in the order help prints them.
var Formats = []string{"text", "json", "md", "xml"}

// Render writes the report in one of Formats. An unknown format is an error
// rather than a fallback to text: a script asking for JSON and getting
// coloured text would parse nothing and report nothing.
func Render(format string, blocks []Block, w io.Writer) error {
	switch format {
	case "", "text":
		renderText(blocks, w)
	case "json":
		return renderJSON(blocks, w)
	case "md", "markdown":
		renderMarkdown(blocks, w)
	case "xml":
		return renderXML(blocks, w)
	default:
		return fmt.Errorf("unknown format %q; expected one of %s", format, strings.Join(Formats, ", "))
	}
	return nil
}

// ── text ──────────────────────────────────────────────────────────────

func renderText(blocks []Block, w io.Writer) {
	for _, b := range blocks {
		switch {
		case b.Rows != nil || len(b.Columns) > 0:
			renderTextTable(b, w)
		case b.Lines != nil:
			for _, line := range b.Lines {
				fmt.Fprintf(w, "\n%s%s: %s%s\n", color.Yellow, b.Title, line, color.Reset)
			}
		default:
			items := make([]fmtc.SectionItem, 0, len(b.Items))
			for _, it := range b.Items {
				items = append(items, fmtc.SectionItem{Key: it.Key, Value: it.Value})
			}
			fmtc.Section(b.Title, items)
		}
	}
}

func renderTextTable(b Block, w io.Writer) {
	widths := make([]int, len(b.Columns))
	for i, c := range b.Columns {
		widths[i] = len(c.Title)
	}
	for _, row := range b.Rows {
		for i := range b.Columns {
			if i < len(row) && len(row[i]) > widths[i] {
				widths[i] = len(row[i])
			}
		}
	}

	fmt.Fprintf(w, "\n%s── %s %s%s\n", color.Cyan, b.Title, strings.Repeat("─", max(0, 40-len(b.Title)-4)), color.Reset)
	if len(b.Rows) == 0 {
		fmt.Fprintf(w, "   %s(none)%s\n", color.Gray, color.Reset)
		return
	}
	fmt.Fprint(w, "   ", color.Gray)
	for i, c := range b.Columns {
		fmt.Fprint(w, padRight(c.Title, widths[i]), "  ")
	}
	fmt.Fprintln(w, color.Reset)
	for _, row := range b.Rows {
		fmt.Fprint(w, "   ")
		for i := range b.Columns {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			fmt.Fprint(w, padRight(cell, widths[i]), "  ")
		}
		fmt.Fprintln(w)
	}
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// ── json ──────────────────────────────────────────────────────────────

// orderedObject keeps block order in the JSON output; encoding/json sorts
// map keys, and a report whose sections come out alphabetised reads worse
// than the text it replaces.
type orderedObject struct {
	keys   []string
	values map[string]any
}

func newOrderedObject() *orderedObject {
	return &orderedObject{values: map[string]any{}}
}

func (o *orderedObject) set(key string, value any) {
	if _, ok := o.values[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

func (o *orderedObject) get(key string) (any, bool) {
	v, ok := o.values[key]
	return v, ok
}

func (o *orderedObject) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		vb, err := json.Marshal(o.values[k])
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func blockValue(b Block) any {
	switch {
	case b.Rows != nil || len(b.Columns) > 0:
		rows := make([]*orderedObject, 0, len(b.Rows))
		for _, row := range b.Rows {
			o := newOrderedObject()
			for i, c := range b.Columns {
				if i < len(row) {
					o.set(c.Key, row[i])
				} else {
					o.set(c.Key, "")
				}
			}
			rows = append(rows, o)
		}
		return rows
	case b.Lines != nil:
		return b.Lines
	default:
		o := newOrderedObject()
		for _, it := range b.Items {
			o.set(it.Key, it.Value)
		}
		return o
	}
}

func renderJSON(blocks []Block, w io.Writer) error {
	root := newOrderedObject()
	for _, b := range blocks {
		if b.Name == "" {
			root.set(b.Key, blockValue(b))
			continue
		}
		group, ok := root.get(b.Key)
		if !ok {
			group = newOrderedObject()
			root.set(b.Key, group)
		}
		group.(*orderedObject).set(b.Name, blockValue(b))
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(root)
}

// ── markdown ──────────────────────────────────────────────────────────

func renderMarkdown(blocks []Block, w io.Writer) {
	for i, b := range blocks {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "## %s\n\n", b.Title)
		switch {
		case b.Rows != nil || len(b.Columns) > 0:
			titles := make([]string, len(b.Columns))
			for i, c := range b.Columns {
				titles[i] = mdCell(c.Title)
			}
			fmt.Fprintf(w, "| %s |\n", strings.Join(titles, " | "))
			fmt.Fprintf(w, "|%s\n", strings.Repeat("---|", len(b.Columns)))
			for _, row := range b.Rows {
				cells := make([]string, len(b.Columns))
				for i := range b.Columns {
					if i < len(row) {
						cells[i] = mdCell(row[i])
					}
				}
				fmt.Fprintf(w, "| %s |\n", strings.Join(cells, " | "))
			}
		case b.Lines != nil:
			for _, line := range b.Lines {
				fmt.Fprintf(w, "- %s\n", line)
			}
		default:
			fmt.Fprintln(w, "| Key | Value |")
			fmt.Fprintln(w, "|---|---|")
			for _, it := range b.Items {
				fmt.Fprintf(w, "| %s | %s |\n", mdCell(it.Key), mdCell(it.Value))
			}
		}
	}
}

func mdCell(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}

// ── xml ───────────────────────────────────────────────────────────────

func renderXML(blocks []Block, w io.Writer) error {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString("<info>\n")

	// Blocks sharing a Key are wrapped once: <scopes><scope name="old">…
	i := 0
	for i < len(blocks) {
		b := blocks[i]
		if b.Name == "" {
			writeXMLBlock(&buf, b, "  ")
			i++
			continue
		}
		fmt.Fprintf(&buf, "  <%s>\n", xmlName(b.Key))
		for i < len(blocks) && blocks[i].Key == b.Key && blocks[i].Name != "" {
			writeXMLBlock(&buf, blocks[i], "    ")
			i++
		}
		fmt.Fprintf(&buf, "  </%s>\n", xmlName(b.Key))
	}

	buf.WriteString("</info>\n")
	_, err := w.Write(buf.Bytes())
	return err
}

func writeXMLBlock(buf *bytes.Buffer, b Block, indent string) {
	tag := xmlName(b.Key)
	if b.Name != "" {
		tag = singular(tag)
		fmt.Fprintf(buf, "%s<%s name=\"%s\">\n", indent, tag, xmlEscape(b.Name))
	} else {
		fmt.Fprintf(buf, "%s<%s>\n", indent, tag)
	}

	switch {
	case b.Rows != nil || len(b.Columns) > 0:
		rowTag := b.RowName
		if rowTag == "" {
			rowTag = "row"
		}
		for _, row := range b.Rows {
			fmt.Fprintf(buf, "%s  <%s>\n", indent, rowTag)
			for i, c := range b.Columns {
				cell := ""
				if i < len(row) {
					cell = row[i]
				}
				fmt.Fprintf(buf, "%s    <%s>%s</%s>\n", indent, xmlName(c.Key), xmlEscape(cell), xmlName(c.Key))
			}
			fmt.Fprintf(buf, "%s  </%s>\n", indent, rowTag)
		}
	case b.Lines != nil:
		for _, line := range b.Lines {
			fmt.Fprintf(buf, "%s  <line>%s</line>\n", indent, xmlEscape(line))
		}
	default:
		for _, it := range b.Items {
			fmt.Fprintf(buf, "%s  <%s>%s</%s>\n", indent, xmlName(it.Key), xmlEscape(it.Value), xmlName(it.Key))
		}
	}

	fmt.Fprintf(buf, "%s</%s>\n", indent, tag)
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// xmlName turns a report key into a well-formed element name: host codes
// carry "#2" suffixes and config keys carry slashes, neither of which an
// XML parser accepts in a tag.
func xmlName(key string) string {
	var b strings.Builder
	for i, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
			b.WriteRune(r)
		case (r >= '0' && r <= '9') || r == '-' || r == '.':
			if i == 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "_"
	}
	return b.String()
}

func singular(tag string) string {
	if strings.HasSuffix(tag, "s") && len(tag) > 1 {
		return tag[:len(tag)-1]
	}
	return tag
}
