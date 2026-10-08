package deprecation

import (
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/version"
)

// TestNothingOutlivesItsMajor is the rule made into a gate: compatibility code
// is deleted in the next major. When version.Version reaches an entry's
// RemoveIn, this fails and names the code — delete it, then the entry.
func TestNothingOutlivesItsMajor(t *testing.T) {
	for _, e := range DueNow() {
		t.Errorf("madock %s is the major that removes this — delete the code in %s, then the ledger entry:\n  %s %s → %s\n  %s",
			version.Version, e.Where, e.Kind, e.Old, e.New, e.What)
	}
}

// An entry that cannot be acted on is worse than none: it looks like a plan.
func TestEveryEntryCanBeActedOn(t *testing.T) {
	for i, e := range Ledger {
		if e.Where == "" || e.What == "" || e.Since == "" {
			t.Errorf("entry %d (%s %s) needs What, Where and Since", i, e.Kind, e.Old)
		}
		if !strings.HasSuffix(e.RemoveIn, ".0.0") {
			t.Errorf("entry %d: RemoveIn %q is not a major release (X.0.0)", i, e.RemoveIn)
		}
		if major(e.RemoveIn) <= major(e.Since) {
			t.Errorf("entry %d: RemoveIn %s is not after Since %s", i, e.RemoveIn, e.Since)
		}
		if e.Kind == RenamedKey && (e.Old == "" || e.New == "") {
			t.Errorf("entry %d: a renamed key needs Old and New", i)
		}
	}
}

func TestDueArrivesWithTheMajor(t *testing.T) {
	saved := Ledger
	t.Cleanup(func() { Ledger = saved })
	Ledger = []Entry{{Kind: Other, What: "x", Where: "y", Since: "4.3.1", RemoveIn: "5.0.0"}}

	if got := Due("4.9.9"); len(got) != 0 {
		t.Errorf("due before its major: %v", got)
	}
	if got := Due("5.0.0"); len(got) != 1 {
		t.Errorf("not due at its major: %v", got)
	}
	if got := Due("v6.1.0"); len(got) != 1 {
		t.Errorf("not due after its major: %v", got)
	}
}
