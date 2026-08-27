package all

import (
	"sort"
	"testing"

	"github.com/faradey/madock/v4/src/command"
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
// every project and "why is it answering 502" is asked from wherever one stands;
// project:list because it describes the installation, and the reason to run it is
// usually that something about the registry is wrong; template:convert because a
// directory of templates is a directory of templates — refusing to run it outside
// a project would be refusing it in exactly the place somebody keeps a copy of
// their overrides; project:orphans because what it reports is precisely what has
// no project directory left to stand in — a volume that outlived the project it
// belonged to; and project:remove because an orphan — a registry entry whose
// source directory is gone — has nowhere to be removed from. project:list has been
// able to name those since 3.8.50 and nothing could act on them: the tool described
// a problem it could not fix. The exemption is narrow in the handler, not here —
// without --name it still requires a project directory, so the protection that
// keeps it from destroying the wrong one is unchanged.
func TestGlobalCommands(t *testing.T) {
	want := []string{
		"--version",
		"-v",
		"help",
		"mcp",
		"project:clone",
		"project:list",
		"project:orphans",
		"project:remove",
		"proxy:logs",
		"setup",
		"template:convert",
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
