package all

import (
	"sort"
	"testing"

	"github.com/faradey/madock/v3/src/command"
)

// TestGlobalCommands pins which commands may run outside a project.
//
// The dispatcher refuses a project command in a directory that is not a project,
// and the project name comes from the directory name — so before that refusal
// existed, `stop` in any same-named directory drove docker compose with whatever
// generated files carried that name. Global is the exemption, and it is the kind of
// flag that gets added in passing: this test makes adding one a deliberate edit in
// two places rather than a one-line decision nobody reviews.
//
// setup and project:clone are here because they create a project; help, version and
// mcp because they answer about the binary; proxy:logs because one proxy serves
// every project and "why is it answering 502" is asked from wherever one stands.
func TestGlobalCommands(t *testing.T) {
	want := []string{
		"--version",
		"-v",
		"help",
		"mcp",
		"project:clone",
		"proxy:logs",
		"setup",
		"version",
	}

	var got []string
	for _, def := range command.GetAll() {
		if def.Global {
			got = append(got, def.Aliases...)
		}
	}
	sort.Strings(got)

	if len(got) != len(want) {
		t.Fatalf("global commands = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("global commands = %v, want %v", got, want)
		}
	}
}
