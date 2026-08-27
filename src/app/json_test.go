package app

import "testing"

// The dispatcher decides whether --json was asked for before any command parses
// its arguments, so it reads the raw argv and has to be exact about it. A false
// positive refuses a command that was never asked for JSON; a false negative is
// the silence this whole change is about.
func TestWantsJSON(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want bool
	}{
		{"the long flag", []string{"--json"}, true},
		{"the short flag", []string{"-q", "-j"}, true},
		{"an explicit value", []string{"--json=true"}, true},
		{"nothing of the sort", []string{"--quiet", "SELECT 1"}, false},
		// A query is an ordinary positional argument and may say anything. Only an
		// exact token counts, or `db:execute "SELECT '--json'"` would be refused
		// for quoting a string.
		{"the word inside a value", []string{"SELECT '--json' AS asked"}, false},
		{"a longer flag that starts the same", []string{"--jsonl"}, false},
		// Everything after -- belongs to whatever the command passes it to, the
		// same rule --help follows.
		{"after a double dash", []string{"--", "--json"}, false},
	}

	for _, c := range cases {
		if got := wantsJSON(c.argv); got != c.want {
			t.Errorf("%s: wantsJSON(%q) = %v, want %v", c.name, c.argv, got, c.want)
		}
	}
}
