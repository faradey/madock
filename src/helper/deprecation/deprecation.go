// Package deprecation is the ledger of what madock still supports only for the
// sake of the past, and the release that takes each one away.
//
// The rule, decided 2026-10-08: compatibility code lives until the next major
// version and goes in it. Writing that down as a rule is not enough — a shim
// nobody remembers is a shim that stays — so every one of them is an entry
// here, and TestNothingOutlivesItsMajor fails the build of the major that is
// due to remove it. Removing an entry is the reminder to remove the code it
// names; the test cannot see the code, only the promise.
//
// What belongs here: an old config name still read, an old template spelling
// still answered, a flag kept as an alias, a file format still parsed. What
// does not: a migration that already ran — a migration is a one-way step,
// not a shim, and it is retired by the same major for the same reason, but
// it is listed under Migration so it is found by the same search.
package deprecation

import (
	"strconv"
	"strings"

	"github.com/faradey/madock/v4/src/version"
)

// Kind says what sort of compatibility an entry keeps.
type Kind string

const (
	// RenamedKey is a config key that moved; Old is still read as New.
	RenamedKey Kind = "renamed-key"
	// Migration is a one-way upgrade step kept for installations that skipped
	// releases.
	Migration Kind = "migration"
	// Other is anything else, described in What and Where.
	Other Kind = "other"
)

// Entry is one thing madock keeps working only for the past.
type Entry struct {
	Kind Kind
	// Old and New, for a renamed key.
	Old, New string
	// What is kept and why, one sentence for whoever removes it.
	What string
	// Where the code to delete lives.
	Where string
	// Since is the release that introduced the compatibility.
	Since string
	// RemoveIn is the major release that deletes it: X.0.0.
	RemoveIn string
}

// Ledger is every entry, oldest first.
var Ledger = []Entry{
	{
		Kind:     RenamedKey,
		Old:      "php/limits/max_execution_time_web",
		New:      "php/limits/max_execution_time_setup",
		What:     "the old name of the setup wizard's time limit is still read from every config layer and still answered to copied templates",
		Where:    "src/helper/configs/renamed.go",
		Since:    "4.3.1",
		RemoveIn: "5.0.0",
	},
}

// RenamedKeys is the old-name → new-name map of every RenamedKey entry.
func RenamedKeys() map[string]string {
	keys := map[string]string{}
	for _, e := range Ledger {
		if e.Kind == RenamedKey {
			keys[e.Old] = e.New
		}
	}
	return keys
}

// Due lists the entries whose major has arrived for a given version.
func Due(current string) []Entry {
	var due []Entry
	for _, e := range Ledger {
		if major(current) >= major(e.RemoveIn) {
			due = append(due, e)
		}
	}
	return due
}

// DueNow is Due for the version this binary reports.
func DueNow() []Entry {
	return Due(version.Version)
}

func major(v string) int {
	head, _, _ := strings.Cut(strings.TrimPrefix(v, "v"), ".")
	n, err := strconv.Atoi(head)
	if err != nil {
		return 0
	}
	return n
}
