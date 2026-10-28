package configs

import (
	"strings"
	"testing"
)

const pruneFixture = `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <platform>custom</platform>
            <note></note>
            <varnish>
                <enabled>false</enabled>
                <config_file>default.vcl</config_file>
            </varnish>
            <cache>
                <redis>
                    <auth>
                        <password>secret</password>
                    </auth>
                </redis>
            </cache>
        </default>
    </scopes>
</config>
`

// Removing every child of a group used to leave the group behind as an empty
// element. It then read back as a value, and every key under it failed to
// assemble with "is already a value".
func TestEditXml_RemovingEveryChildTakesTheGroup(t *testing.T) {
	out, err := editXml([]byte(pruneFixture), nil, []string{"varnish/enabled", "varnish/config_file"}, "default")
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	if strings.Contains(got, "varnish") {
		t.Errorf("the emptied group was left in the file:\n%s", got)
	}
	if _, still := ParseXmlBytes(out)["scopes/default/varnish"]; still {
		t.Error("the emptied group still reads back as a value")
	}
	if !strings.Contains(got, "<platform>custom</platform>") {
		t.Errorf("a neighbour was removed too:\n%s", got)
	}
}

// A group that still has a child is not empty and stays.
func TestEditXml_AGroupWithAChildLeft_Stays(t *testing.T) {
	out, err := editXml([]byte(pruneFixture), nil, []string{"varnish/enabled"}, "default")
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	if !strings.Contains(got, "<config_file>default.vcl</config_file>") {
		t.Errorf("the remaining child was removed:\n%s", got)
	}
	if strings.Contains(got, "<enabled>") {
		t.Errorf("the removed key is still there:\n%s", got)
	}
}

// The group goes up the whole chain that it emptied, and stops where something
// is still in it.
func TestEditXml_EmptiedChainIsTakenUpToTheFirstNonEmpty(t *testing.T) {
	out, err := editXml([]byte(pruneFixture), nil, []string{"cache/redis/auth/password"}, "default")
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	for _, gone := range []string{"<cache>", "<redis>", "<auth>", "password"} {
		if strings.Contains(got, gone) {
			t.Errorf("%q survived though its whole chain was emptied:\n%s", gone, got)
		}
	}
	if !strings.Contains(got, "<varnish>") {
		t.Errorf("an unrelated group was removed:\n%s", got)
	}
}

// An element that was empty to begin with is a value, not a group that lost
// its children, and removing a neighbour must not take it.
func TestEditXml_AnEmptyValueThatWasNeverAGroupStays(t *testing.T) {
	out, err := editXml([]byte(pruneFixture), nil, []string{"platform"}, "default")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "<note></note>") {
		t.Errorf("an empty value was taken for an emptied group:\n%s", out)
	}
}
