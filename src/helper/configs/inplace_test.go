package configs

import (
	"strings"
	"testing"
)

const handWritten = `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <!-- The database is off on purpose: this app talks to the shared
                 cluster, and a second one would shadow it. -->
            <platform>custom</platform>
            <language>nodejs</language>
            <db>
                <enabled>false</enabled>
            </db>

            <cron>
                <enabled>true</enabled>
                <jobs>
                    <apply_due>
                        <schedule>* * * * *</schedule>
                    </apply_due>
                </jobs>
            </cron>
        </default>
    </scopes>
</config>
`

// The whole point. A comment in this file is usually the record of *why* a
// setting is what it is, which the values themselves cannot say — and rendering
// a parsed map loses every one of them.
func TestEditXml_KeepsCommentsAndOrder(t *testing.T) {
	out, err := editXml([]byte(handWritten), map[string]string{"language": "php"}, nil, "default")
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	if !strings.Contains(got, "a second one would shadow it") {
		t.Errorf("the comment was lost:\n%s", got)
	}
	if !strings.Contains(got, "<language>php</language>") {
		t.Errorf("the value was not changed:\n%s", got)
	}
	if strings.Index(got, "<platform>") > strings.Index(got, "<language>") {
		t.Errorf("the keys were reordered:\n%s", got)
	}
	// Everything else is byte-for-byte what it was.
	if want := strings.Replace(handWritten, "<language>nodejs</language>", "<language>php</language>", 1); got != want {
		t.Errorf("something other than the value changed:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// A key the file does not have yet goes inside the parent that does, indented
// like its neighbours.
func TestEditXml_AddsIntoAnExistingParent(t *testing.T) {
	out, err := editXml([]byte(handWritten), map[string]string{"db/type": "mysql"}, nil, "default")
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	if !strings.Contains(got, "<type>mysql</type>") {
		t.Errorf("the key was not added:\n%s", got)
	}
	if !strings.Contains(got, "                <enabled>false</enabled>") {
		t.Errorf("the sibling lost its indentation:\n%s", got)
	}
	if !strings.Contains(got, "                <type>mysql</type>") {
		t.Errorf("the new key is not indented like its siblings:\n%s", got)
	}
	if !strings.Contains(got, "a second one would shadow it") {
		t.Errorf("the comment was lost while adding a key:\n%s", got)
	}
	// It has to land inside <db>, not after it.
	dbClose := strings.Index(got, "</db>")
	if strings.Index(got, "<type>mysql</type>") > dbClose {
		t.Errorf("the key landed outside its parent:\n%s", got)
	}
}

// A key whose parents do not exist at all brings them with it.
func TestEditXml_CreatesMissingParents(t *testing.T) {
	out, err := editXml([]byte(handWritten), map[string]string{"nodejs/embedded/enabled": "true"}, nil, "default")
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	for _, want := range []string{"<nodejs>", "<embedded>", "<enabled>true</enabled>", "</embedded>", "</nodejs>"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q is missing:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "a second one would shadow it") {
		t.Errorf("the comment was lost:\n%s", got)
	}

	// And the result still parses to the value that was asked for.
	if got := ParseXmlBytes(out)["scopes/default/nodejs/embedded/enabled"]; got != "true" {
		t.Errorf("the created key reads back as %q", got)
	}
}

// Setting a key that has children must not erase them. "db" is a parent here,
// and writing a value into it would take <enabled> with it.
func TestEditXml_RefusesToFlattenAParent(t *testing.T) {
	out, err := editXml([]byte(handWritten), map[string]string{"db": "nonsense"}, nil, "default")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(out), "<enabled>false</enabled>") {
		t.Errorf("a parent's children were erased:\n%s", out)
	}
}

// Nothing to do means no diff, which is what keeps an upgrade from leaving a
// change in somebody's repository for no reason.
func TestEditXml_NoChangeNoDiff(t *testing.T) {
	out, err := editXml([]byte(handWritten), map[string]string{"language": "nodejs"}, nil, "default")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != handWritten {
		t.Errorf("the file was rewritten for a value it already had:\n%s", out)
	}
}

// Several changes at once, including one that inserts — the offsets of the
// earlier edits have to survive the later ones.
func TestEditXml_SeveralAtOnce(t *testing.T) {
	out, err := editXml([]byte(handWritten), map[string]string{
		"language":   "php",
		"db/enabled": "true",
		"db/type":    "mysql",
	}, nil, "default")
	if err != nil {
		t.Fatal(err)
	}

	parsed := ParseXmlBytes(out)
	for key, want := range map[string]string{
		"scopes/default/language":   "php",
		"scopes/default/db/enabled": "true",
		"scopes/default/db/type":    "mysql",
	} {
		if got := parsed[key]; got != want {
			t.Errorf("%s = %q, want %q\n%s", key, got, want, out)
		}
	}
	if !strings.Contains(string(out), "a second one would shadow it") {
		t.Errorf("the comment was lost:\n%s", out)
	}
	if !strings.Contains(string(out), "<schedule>* * * * *</schedule>") {
		t.Errorf("an untouched branch was disturbed:\n%s", out)
	}
}

// A scope this file does not carry is not a reason to write anything.
func TestEditXml_UnknownScopeChangesNothing(t *testing.T) {
	out, err := editXml([]byte(handWritten), map[string]string{"language": "php"}, nil, "staging")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != handWritten {
		t.Errorf("a scope that is not in the file was written into:\n%s", out)
	}
}

// Deleting is the half that never existed anywhere in madock. A setting taken
// out of a project's config stayed in the installed copy for good: config:set
// can only assign, there is no unset, and clearing the cache does not touch the
// file. The only way to drop one key was to remove the project and set it up
// again.
func TestEditXml_Removes(t *testing.T) {
	out, err := editXml([]byte(handWritten), nil, []string{"language"}, "default")
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	if strings.Contains(got, "<language>") {
		t.Errorf("the setting is still there:\n%s", got)
	}
	if !strings.Contains(got, "a second one would shadow it") {
		t.Errorf("the comment was lost while removing:\n%s", got)
	}
	if !strings.Contains(got, "<platform>custom</platform>") {
		t.Errorf("a neighbour was removed too:\n%s", got)
	}
	// No blank, indented line left where it was.
	if strings.Contains(got, "\n            \n") {
		t.Errorf("an empty indented line was left behind:\n%q", got)
	}
	if _, still := ParseXmlBytes(out)["scopes/default/language"]; still {
		t.Error("the key still parses out of the file")
	}
}

// Removing a branch takes its children, which is what an explicit unset of a
// branch means.
func TestEditXml_RemovesABranch(t *testing.T) {
	out, err := editXml([]byte(handWritten), nil, []string{"cron"}, "default")
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	for _, gone := range []string{"<cron>", "<jobs>", "apply_due", "<schedule>"} {
		if strings.Contains(got, gone) {
			t.Errorf("%q survived the removal of its branch:\n%s", gone, got)
		}
	}
	if !strings.Contains(got, "<db>") {
		t.Errorf("an unrelated branch was removed:\n%s", got)
	}
}

// Removing and setting in one pass: the offsets of each have to survive the
// other.
func TestEditXml_RemovesAndSetsAtOnce(t *testing.T) {
	out, err := editXml([]byte(handWritten), map[string]string{"platform": "magento2"}, []string{"language"}, "default")
	if err != nil {
		t.Fatal(err)
	}

	parsed := ParseXmlBytes(out)
	if got := parsed["scopes/default/platform"]; got != "magento2" {
		t.Errorf("platform = %q\n%s", got, out)
	}
	if _, still := parsed["scopes/default/language"]; still {
		t.Errorf("language survived:\n%s", out)
	}
	if !strings.Contains(string(out), "a second one would shadow it") {
		t.Errorf("the comment was lost:\n%s", out)
	}
}

// A key that is not in the file is not an error, and not a reason to rewrite it.
func TestEditXml_RemovingWhatIsNotThere(t *testing.T) {
	out, err := editXml([]byte(handWritten), nil, []string{"search/meilisearch/enabled"}, "default")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != handWritten {
		t.Errorf("the file was rewritten for a key it does not have:\n%s", out)
	}
}

// The inserted setting has to look like it was typed there. Checked as exact
// bytes, because the Contains assertions above passed while the writer was
// leaving a blank, indented line above every insertion.
func TestEditXml_InsertionIsExact(t *testing.T) {
	out, err := editXml([]byte(handWritten), map[string]string{"db/type": "mysql"}, nil, "default")
	if err != nil {
		t.Fatal(err)
	}

	want := strings.Replace(handWritten,
		"            <db>\n                <enabled>false</enabled>\n            </db>",
		"            <db>\n                <enabled>false</enabled>\n                <type>mysql</type>\n            </db>", 1)

	if string(out) != want {
		t.Errorf("insertion is not byte-exact:\n--- got ---\n%q\n--- want ---\n%q", out, want)
	}
}
