package configs

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"sort"
	"strings"
)

// SaveKeepingComments writes the given keys into an existing config file by
// editing its text, rather than by parsing it into a map and rendering the map
// back.
//
// The difference matters for exactly one file: a project's own
// `.madock/config.xml`. That one is written by a person and committed to their
// repository, and its comments are usually the record of *why* a setting is
// what it is — "the database is off because this app talks to the shared
// cluster" is not something the values can say. Rendering a parsed map loses
// every one of them, reorders the keys alphabetically, and turns a one-line
// change into a diff nobody reads.
//
// Everything else madock writes — the registry copies, the installation's own
// config — is machine-owned, has no comments, and goes on using SaveInFile.
// This is deliberately not a general replacement: the renderer is used by
// `config:set`, by setup and by the password bootstrap, and swapping all of
// that at once would put every config file on this path for the benefit of one.
//
// The editing is byte-level. Each token's offset comes from the decoder, so an
// element whose value changes has only the text between its tags replaced;
// every other byte of the file is copied through untouched. Keys that are not
// in the file yet are inserted before the closing tag of the nearest parent
// that does exist, indented to match its children.
//
// Keys are given scope-relative, as SaveInFile takes them: "db/enabled", not
// "scopes/default/db/enabled".
func SaveKeepingComments(file string, data map[string]string, activeScope string) error {
	return editFile(file, data, nil, activeScope)
}

// RemoveKeepingComments deletes settings from a config file, leaving the rest
// of the document as it was.
//
// Deleting is the half that never existed. A setting taken out of a project's
// config stayed in the installed copy for good: `config:set` can only assign,
// there is no unset, and clearing the cache does not touch the file — so the
// only way to drop one key was to remove the project and set it up again. A
// team that deletes a setting, commits and rolls out therefore leaves every
// machine that ever ran setup living by the old value, with nothing failing and
// nobody told.
//
// Removing a key that has children removes them with it, which is what an
// explicit unset of a branch means.
func RemoveKeepingComments(file string, keys []string, activeScope string) error {
	return editFile(file, nil, keys, activeScope)
}

func editFile(file string, data map[string]string, remove []string, activeScope string) error {
	body, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	edited, err := editXml(body, data, remove, activeScope)
	if err != nil {
		return err
	}
	if bytes.Equal(edited, body) {
		// Nothing moved. Not writing at all is what keeps an upgrade from
		// leaving a diff in somebody's repository for no reason.
		return nil
	}

	return os.WriteFile(file, edited, ConfigFilePermissions)
}

// edit is one change to make to the document.
type edit struct {
	start, end int    // byte range to replace
	with       string // what to put there
}

// editXml applies the changes to the document text and returns the result.
func editXml(body []byte, data map[string]string, remove []string, activeScope string) ([]byte, error) {
	if len(data) == 0 && len(remove) == 0 {
		return body, nil
	}

	dropping := make(map[string]bool, len(remove))
	for _, key := range remove {
		dropping[key] = true
	}

	prefix := []string{"config", "scopes", activeScope}

	remaining := make(map[string]string, len(data))
	for key, value := range data {
		remaining[key] = value
	}

	var edits []edit

	// Where each existing element ends, by its scope-relative path. Used to
	// place a new leaf inside a parent that is already there.
	closeOf := map[string]int{}
	indentOf := map[string]string{}
	openedAt := map[string]int{}

	decoder := xml.NewDecoder(bytes.NewReader(body))
	var stack []string
	var lastText string
	valueStart := -1

	for {
		before := decoder.InputOffset()
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		after := decoder.InputOffset()

		switch t := token.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name.Local)
			valueStart = int(after)
			lastText = ""

			if rel, ok := relativeTo(stack, prefix); ok {
				indentOf[rel] = indentBefore(body, int(before))
				openedAt[rel] = int(before)
			} else if isScopeItself(stack, prefix) {
				// The scope element is the insertion point of last resort: a
				// key whose whole parent chain is missing has nowhere else to
				// go. relativeTo deliberately does not report it — its path
				// relative to itself is empty — so it is recorded here.
				indentOf[""] = indentBefore(body, int(before))
			}

		case xml.CharData:
			lastText = string(t)

		case xml.EndElement:
			if isScopeItself(stack, prefix) {
				closeOf[""] = int(before)
			}

			rel, inScope := relativeTo(stack, prefix)
			if inScope {
				closeOf[rel] = int(before)

				if dropping[rel] {
					// The whole element goes, with the line it sits on and the
					// indentation in front of it — otherwise a blank, indented
					// line is left where the setting was.
					edits = append(edits, edit{lineStart(body, openedAt[rel]), int(after), ""})
				}

				if value, wanted := remaining[rel]; wanted {
					// A leaf that is already here: replace what is between the
					// tags and nothing else. A parent element is skipped —
					// setting "db" when the file has <db><enabled> would erase
					// its children.
					if isLeaf(lastText, body, valueStart, int(before)) {
						edits = append(edits, edit{valueStart, int(before), value})
						delete(remaining, rel)
					}
				}
			}
			stack = stack[:len(stack)-1]
			valueStart = -1
		}
	}

	// What is left has to be created. Deepest-first is deliberate: creating
	// "db/type" may need <db>, and if both are missing the same insertion point
	// serves them, so they are grouped by the nearest parent that exists.
	for _, key := range sortedKeys(remaining) {
		at, indent, ok := insertionFor(key, closeOf, indentOf)
		if !ok {
			continue // no scope in this file to write into
		}
		// The whitespace already sitting in front of the closing tag is
		// replaced rather than kept: leaving it produces a blank, indented line
		// above every inserted setting.
		edits = append(edits, edit{trimWhitespaceBack(body, at), at, renderMissing(key, remaining[key], at, closeOf, indent)})
	}

	return applyEdits(body, edits), nil
}

// isScopeItself reports whether the current element is the scope element.
func isScopeItself(stack, prefix []string) bool {
	if len(stack) != len(prefix) {
		return false
	}
	for i, name := range prefix {
		if stack[i] != name {
			return false
		}
	}
	return true
}

// relativeTo reports the path of the current element inside the active scope.
func relativeTo(stack, prefix []string) (string, bool) {
	if len(stack) <= len(prefix) {
		return "", false
	}
	for i, name := range prefix {
		if stack[i] != name {
			return "", false
		}
	}
	return strings.Join(stack[len(prefix):], "/"), true
}

// isLeaf reports whether the element that just closed holds text rather than
// other elements.
func isLeaf(text string, body []byte, from, to int) bool {
	if from < 0 || to < from || to > len(body) {
		return false
	}
	return !bytes.Contains(body[from:to], []byte("<"))
}

// trimWhitespaceBack walks back over the whitespace that ends the text before
// an offset, so an insertion can lay down its own.
func trimWhitespaceBack(body []byte, at int) int {
	i := at
	for i > 0 && (body[i-1] == ' ' || body[i-1] == '\t' || body[i-1] == '\n' || body[i-1] == '\r') {
		i--
	}
	return i
}

// lineStart is the offset of the beginning of the line an element opens on, so
// removing it takes its indentation with it.
func lineStart(body []byte, at int) int {
	if at <= 0 || at > len(body) {
		return at
	}
	start := bytes.LastIndexByte(body[:at], '\n')
	if start < 0 {
		return 0
	}
	if len(bytes.TrimLeft(body[start+1:at], " \t")) != 0 {
		return at // something else shares the line; leave it alone
	}
	return start
}

// indentBefore returns the whitespace that starts the line an element sits on,
// so anything inserted beside it lines up.
func indentBefore(body []byte, at int) string {
	start := bytes.LastIndexByte(body[:at], '\n') + 1
	indent := body[start:at]
	if len(bytes.TrimLeft(indent, " \t")) != 0 {
		return ""
	}
	return string(indent)
}

// insertionFor finds the nearest existing ancestor of a key and returns the
// offset just before that ancestor's closing tag.
func insertionFor(key string, closeOf map[string]int, indentOf map[string]string) (int, string, bool) {
	parts := strings.Split(key, "/")
	for i := len(parts) - 1; i > 0; i-- {
		parent := strings.Join(parts[:i], "/")
		if at, ok := closeOf[parent]; ok {
			return at, indentOf[parent] + "    ", true
		}
	}
	// The scope itself, which relativeTo records as the empty path.
	if at, ok := closeOf[""]; ok {
		return at, indentOf[""] + "    ", true
	}
	return 0, "", false
}

// renderMissing writes the elements a key needs that the file does not have.
func renderMissing(key, value string, at int, closeOf map[string]int, indent string) string {
	parts := strings.Split(key, "/")

	// How much of the path already exists decides how much has to be created.
	have := 0
	for i := len(parts) - 1; i > 0; i-- {
		if _, ok := closeOf[strings.Join(parts[:i], "/")]; ok {
			have = i
			break
		}
	}

	missing := parts[have:]
	var out strings.Builder
	for i, name := range missing {
		out.WriteString("\n" + indent + strings.Repeat("    ", i) + "<" + name + ">")
		if i == len(missing)-1 {
			out.WriteString(value + "</" + name + ">")
		}
	}
	for i := len(missing) - 2; i >= 0; i-- {
		out.WriteString("\n" + indent + strings.Repeat("    ", i) + "</" + missing[i] + ">")
	}
	out.WriteString("\n" + strings.TrimSuffix(indent, "    "))

	return out.String()
}

// applyEdits rewrites the document, last change first so earlier offsets stay
// valid.
func applyEdits(body []byte, edits []edit) []byte {
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })

	out := body
	for _, e := range edits {
		if e.start < 0 || e.end > len(out) || e.start > e.end {
			continue
		}
		next := make([]byte, 0, len(out)+len(e.with))
		next = append(next, out[:e.start]...)
		next = append(next, e.with...)
		next = append(next, out[e.end:]...)
		out = next
	}

	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
