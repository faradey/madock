package cli

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// The report this exists for: the parentheses reached bash instead of node,
// because an argument without a space and without an `=` was passed through
// raw.
//
//	madock cli node -e "console.log(process.version)"
//	bash: -c: line 1: syntax error near unexpected token `('
func TestNormalizeCliCommand_KeepsShellSyntaxAsData(t *testing.T) {
	got := NormalizeCliCommandWithJoin([]string{"node", "-e", "console.log(process.version)"})

	want := `'node' '-e' 'console.log(process.version)'`
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

// Every one of these was passed through untouched before, and each is syntax to
// a shell rather than data.
func TestNormalizeCliCommand_QuotesEveryMetacharacter(t *testing.T) {
	for _, arg := range []string{
		"a(b)", "a|b", "a;b", "a&b", "a*b", "a>b", "a<b",
		"$HOME", "`id`", "a\nb", "a'b", `a"b`, "a\\b", "#comment", "~root",
	} {
		out := NormalizeCliCommand([]string{"echo", arg})
		if out[1] == arg {
			t.Errorf("%q was passed to the shell unquoted", arg)
		}
	}
}

// An environment prefix has to keep working: quoting the whole word would stop
// bash reading it as an assignment at all.
func TestNormalizeCliCommand_AssignmentKeepsItsShape(t *testing.T) {
	got := NormalizeCliCommandWithJoin([]string{"APP_ENV=dev with space", "php", "bin/console"})

	// Everything else is quoted, including the command name — a quoted word is
	// still looked up as a command, so there is no reason for a second rule
	// about which arguments are safe.
	want := `APP_ENV='dev with space' 'php' 'bin/console'`
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

// Something that only looks like an assignment is not one — the name has to be
// a valid shell name, or the whole word is just an argument.
func TestNormalizeCliCommand_OnlyRealAssignmentsAreSpared(t *testing.T) {
	got := NormalizeCliCommandWithJoin([]string{"echo", "1=2", "a-b=c"})

	want := `'echo' '1=2' 'a-b=c'`
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

// One argument is a script, several are argv. `madock cli "ls | grep x"` is how
// a pipeline has always been run, and quoting it would turn the pipe into text.
func TestNormalizeCliCommand_SingleArgumentIsAScript(t *testing.T) {
	got := NormalizeCliCommandWithJoin([]string{"ls -la | grep conf"})

	if got != "ls -la | grep conf" {
		t.Fatalf("a single argument must be passed through as a script, got %s", got)
	}
}

// The old code trimmed each argument and stripped quote characters from its
// ends, which silently edited data on its way to a program.
func TestNormalizeCliCommand_DoesNotEditArguments(t *testing.T) {
	got := NormalizeCliCommand([]string{"echo", `  "padded"  `})

	if got[1] != `'  "padded"  '` {
		t.Fatalf("the argument was edited on the way through: %s", got[1])
	}
}

// The test that answers the actual question: does the string survive a shell?
// Everything above is about the shape of the output; this runs it.
func TestNormalizeCliCommand_SurvivesTheShell(t *testing.T) {
	cases := [][]string{
		{"echo", "console.log(process.version)"},
		{"echo", "a|b", "c;d"},
		{"echo", "$HOME"},
		{"echo", "it's"},
		{"echo", `say "hi"`},
		{"echo", "*"},
	}

	for _, argv := range cases {
		line := NormalizeCliCommandWithJoin(slices.Clone(argv))

		out, err := exec.Command("bash", "-c", line).Output()
		if err != nil {
			t.Errorf("bash -c %q failed: %v", line, err)
			continue
		}

		want := strings.Join(argv[1:], " ")
		if strings.TrimRight(string(out), "\n") != want {
			t.Errorf("bash -c %q printed %q, want %q", line, strings.TrimRight(string(out), "\n"), want)
		}
	}
}

// A command that must not run. Unquoted, the substitution executes and its
// output becomes the argument.
func TestNormalizeCliCommand_DoesNotRunASubstitution(t *testing.T) {
	line := NormalizeCliCommandWithJoin([]string{"echo", "$(id -u)"})

	out, err := exec.Command("bash", "-c", line).Output()
	if err != nil {
		t.Fatalf("bash -c %q failed: %v", line, err)
	}

	if strings.TrimRight(string(out), "\n") != "$(id -u)" {
		t.Fatalf("the substitution ran: %q", string(out))
	}
}
