package configs

import (
	"strings"
	"testing"
)

// A tag repeated with text inside is the documented way to write cron jobs, and
// xml2map hands it over as []string — a type ComposeConfigMap did not know. The
// type switch matched nothing and the key was dropped without a word, so a
// <jobs> block with one <job> parsed and a block with several parsed to nothing.
//
// Measured on Pricesmith, live and demo, on 2026-08-19: seven jobs in the
// config, cron started, and an empty crontab in the container.
func TestRepeatedTagKeepsEveryEntry(t *testing.T) {
	const doc = `<?xml version="1.0"?>
<config><default><cron><enabled>true</enabled><jobs>
<job>* * * * * /a.sh</job>
<job>* * * * * /b.sh</job>
<job>* * * * * /c.sh</job>
</jobs></cron></default></config>`

	flat := parseFlat(t, doc)

	for key, want := range map[string]string{
		"default/cron/enabled":    "true",
		"default/cron/jobs/job/0": "* * * * * /a.sh",
		"default/cron/jobs/job/1": "* * * * * /b.sh",
		"default/cron/jobs/job/2": "* * * * * /c.sh",
	} {
		if flat[key] != want {
			t.Errorf("%s = %q, want %q", key, flat[key], want)
		}
	}
	if len(flat) != 4 {
		t.Errorf("got %d keys, want 4: %v", len(flat), flat)
	}
}

// One entry has no index — there is nothing in the document to tell a single
// repeated tag from a plain one — and consumers have to accept both spellings.
func TestSingleTagStaysUnindexed(t *testing.T) {
	const doc = `<?xml version="1.0"?>
<config><default><cron><jobs><job>* * * * * /a.sh</job></jobs></cron></default></config>`

	flat := parseFlat(t, doc)
	if flat["default/cron/jobs/job"] != "* * * * * /a.sh" {
		t.Errorf("default/cron/jobs/job = %q", flat["default/cron/jobs/job"])
	}
}

// The writer has to put a list back the way it was read. Spelling the index as
// an element — <job><0>…</0></job> — round-trips through the encoder happily and
// then fails the next read with "invalid XML name: 0"; ParseXmlFile is fatal, so
// that would take the whole command down on a config madock wrote itself.
func TestRepeatedTagSurvivesRoundTrip(t *testing.T) {
	const doc = `<?xml version="1.0"?>
<config><default><cron><enabled>true</enabled><jobs>
<job>* * * * * /a.sh</job>
<job>* * * * * /b.sh</job>
</jobs></cron></default></config>`

	flat := parseFlat(t, doc)

	wide := make(map[string]interface{}, len(flat))
	for key, value := range flat {
		wide[key] = value
	}
	rendered := RenderXml(wide)

	if strings.Contains(rendered, "<0>") {
		t.Fatalf("index written as an element, which is not a legal XML name:\n%s", rendered)
	}
	if want := "<job>* * * * * /a.sh</job>"; !strings.Contains(rendered, want) {
		t.Errorf("rendered config does not contain %s:\n%s", want, rendered)
	}

	back := parseFlat(t, rendered)
	if len(back) != len(flat) {
		t.Fatalf("key count changed on round trip: %d -> %d\n%v", len(flat), len(back), back)
	}
	for key, want := range flat {
		if back[key] != want {
			t.Errorf("round trip lost %s: %q, want %q", key, back[key], want)
		}
	}
}

// A map whose keys are not a complete 0..n-1 run of indices is an ordinary
// branch and must be written as one.
func TestNumericLookalikesAreNotCollapsed(t *testing.T) {
	for name, input := range map[string]map[string]interface{}{
		"gap":         {"a/0": "x", "a/2": "y"},
		"not-numeric": {"a/0": "x", "a/one": "y"},
		"branch":      {"a/0/b": "x", "a/1/b": "y"},
	} {
		nested := SetXmlMap(input)
		if _, isList := nested["a"].([]string); isList {
			t.Errorf("%s: collapsed to a list, want a branch", name)
		}
	}
}

func parseFlat(t *testing.T, doc string) map[string]string {
	t.Helper()
	raw, err := GetXmlMapFromBytes([]byte(doc))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	root, ok := raw["config"].(map[string]interface{})
	if !ok {
		t.Fatalf("no <config> root in %v", raw)
	}
	return ComposeConfigMap(root)
}
