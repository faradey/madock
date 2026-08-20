package app

import "testing"

// TestWantsHelp covers the dispatcher's one job here: telling "explain yourself"
// apart from "do it".
//
// The bug it exists for: `madock install --help` on an installed project printed
// the assembled `bin/magento setup:install …` line — admin password included —
// and ran it over the live database. `install` never called the argument parser,
// so nothing in the process ever looked at the flag. The check moved out of the
// commands and into the one place they all pass through; this is that check.
func TestWantsHelp(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want bool
	}{
		{"nothing", nil, false},
		{"the flag alone", []string{"--help"}, true},
		{"after another flag", []string{"--yes", "--help"}, true},
		{"after a positional", []string{"php", "--help"}, true},

		// -h is not help here, and cannot be: `setup:env -h <host>` uses the
		// short form for --host, and the dispatcher does not know a command's
		// own flags. go-arg still answers -h from inside a command that has an
		// argument struct.
		{"the short form is left alone", []string{"-h"}, false},
		{"the short form with a value", []string{"-h", "example.com"}, false},

		// Everything after -- belongs to whatever the command hands it to.
		{"after a separator", []string{"--", "--help"}, false},
		{"before a separator", []string{"--help", "--", "x"}, true},

		{"a lookalike", []string{"--helpful"}, false},
		{"help as a value", []string{"--name", "--help"}, true}, // deliberate: see below
	}

	for _, c := range cases {
		if got := wantsHelp(c.argv); got != c.want {
			t.Errorf("%s: wantsHelp(%q) = %v, want %v", c.name, c.argv, got, c.want)
		}
	}
}

// The last case above is the known limitation, written down rather than left to
// be discovered: a flag whose *value* is literally "--help" is read as a request
// for help, because the dispatcher does not know which flags take values. It
// costs a printed help page instead of a run, which is the direction this whole
// change chose on purpose — and no madock command has a value that could
// plausibly be "--help".
func TestWantsHelpStopsAtTheSeparator(t *testing.T) {
	if wantsHelp([]string{"--", "--help", "--help"}) {
		t.Error("nothing after -- is the dispatcher's to read")
	}
}
