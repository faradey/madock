package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestQuote(t *testing.T) {
	cases := map[string]string{
		"plain":  `'plain'`,
		"":       `''`,
		"a b":    `'a b'`,
		`it's`:   `'it'\''s'`,
		`'`:      `''\'''`,
		`a'b'c`:  `'a'\''b'\''c'`,
		"100%":   `'100%'`,
		`$(id)`:  `'$(id)'`,
		"back\\": `'back\'`,
		`"x"`:    `'"x"'`,
	}

	for in, want := range cases {
		if got := Quote(in); got != want {
			t.Errorf("Quote(%q) = %s, want %s", in, got, want)
		}
	}
}

// runQuoted hands a quoted value to a real /bin/sh and returns what the shell
// passed on as a single argument. Reading the string proves nothing about how
// the shell will split it.
func runQuoted(t *testing.T, value string) string {
	t.Helper()

	out, err := exec.Command("/bin/sh", "-c", "printf '%s' "+Quote(value)).Output()
	if err != nil {
		t.Fatalf("the shell refused %s: %v", Quote(value), err)
	}
	return string(out)
}

func TestQuote_SurvivesTheShell(t *testing.T) {
	for _, value := range []string{
		"plain",
		"a b",
		`it's`,
		`pa'ss'word`,
		"100%",
		`$(id)`,
		"$HOME",
		"`id`",
		`a"b`,
		"semi;colon",
		"pipe|bar",
		"new\nline",
	} {
		if got := runQuoted(t, value); got != value {
			t.Errorf("value %q came back as %q", value, got)
		}
	}
}

func TestQuote_DoesNotRunACommand(t *testing.T) {
	// The value that decides it: unquoted, this closes the string and appends
	// a command to whatever it was embedded in.
	marker := filepath.Join(t.TempDir(), "executed")
	value := `x'; touch ` + marker + `; echo '`

	got := runQuoted(t, value)

	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the value was executed as a command instead of passed as data")
	}
	if got != value {
		t.Errorf("value came back as %q", got)
	}
}

func TestQuote_EmptyIsStillAnArgument(t *testing.T) {
	// An unset password must arrive as an empty argument rather than
	// disappearing and shifting every argument after it up by one.
	out, err := exec.Command("/bin/sh", "-c", "printf '%s\\n' "+Quote("")+" second").Output()
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) != 2 || lines[0] != "" || lines[1] != "second" {
		t.Errorf("got %q, want an empty argument followed by \"second\"", lines)
	}
}
