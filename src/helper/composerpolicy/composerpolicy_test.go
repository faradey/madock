package composerpolicy

import (
	"strings"
	"testing"
)

// Ids only: the value ends up in a shell command, and accepting advisories
// goes one by one — a word like "all" or anything with a quote is refused.
func TestOnlyAdvisoryIDsAreAccepted(t *testing.T) {
	ids, err := Parse("PKSA-w9tt-7782-78jx, PKSA-db8d-773v-rd1n CVE-2026-12345 GHSA-abcd-efgh-ijkl")
	if err != nil || len(ids) != 4 {
		t.Fatalf("got %v, %v", ids, err)
	}
	for _, bad := range []string{"all", "*", "PKSA-w9tt-7782-78jx'; rm -rf /; '", "league/flysystem"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

func TestTheCommandNamesExactlyTheIDs(t *testing.T) {
	got := ConfigCommand([]string{"PKSA-w9tt-7782-78jx", "PKSA-db8d-773v-rd1n"})
	want := `composer config audit.ignore --json '["PKSA-w9tt-7782-78jx","PKSA-db8d-773v-rd1n"]'`
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
	if ConfigCommand(nil) != "" || strings.Contains(ConfigCommand(nil), "composer") {
		t.Error("no ids must give no command")
	}
}
