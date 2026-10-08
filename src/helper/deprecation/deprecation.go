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
	// TemplateOnly answers the old name to templates without reading a value
	// stored under it as the new key — for a key whose meaning widened in the
	// move, where a value set for the narrow meaning must not become the wide
	// one.
	TemplateOnly bool
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
		Old:      "php/limits/memory",
		New:      "php/ini/memory_limit",
		What:     "memory_limit under its old madock name is still read from every config layer and still answered to copied templates",
		Where:    "src/helper/configs/renamed.go",
		Since:    "4.3.1",
		RemoveIn: "5.0.0",
	},
	{
		Kind:     RenamedKey,
		Old:      "php/limits/max_execution_time",
		New:      "php/ini/max_execution_time",
		What:     "max_execution_time under its old madock name is still read from every config layer and still answered to copied templates",
		Where:    "src/helper/configs/renamed.go",
		Since:    "4.3.1",
		RemoveIn: "5.0.0",
	},
	{
		// /setup's own limit was folded into max_execution_time in 4.3.1, and
		// the old name was dropped outright — while a production store's own
		// copy of the Magento vhost still read it, and rendered
		// max_execution_time= with no value. Found by the unknown-settings
		// warning on that store's first rebuild after the upgrade.
		//
		// Template-only on purpose: a value stored under the old key was set
		// for /setup alone, and read as max_execution_time it would become the
		// limit of every request — 60 seconds for the wizard turning into 60
		// seconds for the store.
		Kind:         RenamedKey,
		Old:          "php/limits/max_execution_time_web",
		New:          "php/ini/max_execution_time",
		TemplateOnly: true,
		What:         "copied templates that still ask for the setup wizard's old time-limit key are answered with max_execution_time; a value stored under the old key is not read",
		Where:        "src/helper/configs/renamed.go",
		Since:        "4.3.2",
		RemoveIn:     "5.0.0",
	},
	{
		Kind:     Other,
		What:     "templates in the pre-3.10 syntax ({{{nginx/port}}}, <<<if, {{{include}}}) are converted at render time; a project's own copies under .madock/docker are the only ones left — on one developer machine on 2026-10-08, seven of ten override templates still used it",
		Where:    "src/helper/tmpl/legacy.go, and its callers in tmpl.Renderer.source and the template audit",
		Since:    "3.10.0",
		RemoveIn: "5.0.0",
	},
}

// RenamedKeys is the old-name → new-name map of the renamed keys whose stored
// values are read under the new name.
func RenamedKeys() map[string]string {
	keys := map[string]string{}
	for _, e := range Ledger {
		if e.Kind == RenamedKey && !e.TemplateOnly {
			keys[e.Old] = e.New
		}
	}
	return keys
}

// TemplateNames is the old-name → new-name map of every renamed key, for
// answering copied templates and for pointing config:set at the new name.
func TemplateNames() map[string]string {
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
