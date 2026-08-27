package all

import (
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/command"
)

// `--json` is declared once, on the argument struct every command embeds, so
// every command accepts it and until 4.1.5 almost none of them did anything with
// it. The dispatcher refuses it now where it is not implemented, and that makes
// `JSONOutput` a mark somebody has to remember on a command that formats JSON —
// exactly the kind of thing that is forgotten silently.
//
// This is the check that notices. It runs against the whole registry, which is
// why it lives in the package that imports every controller.
func TestEveryCommandAdvertisingJSONImplementsIt(t *testing.T) {
	for _, def := range command.GetAll() {
		if len(def.Aliases) == 0 {
			continue
		}
		advertised := strings.Contains(def.Help, "--json")

		if advertised && !def.JSONOutput {
			t.Errorf("%s says it supports --json and is not marked JSONOutput, so the dispatcher refuses the flag it documents", def.Aliases[0])
		}
		if def.JSONOutput && !advertised {
			t.Errorf("%s answers in JSON and does not say so in its help, which is the only place anyone can find out", def.Aliases[0])
		}
	}
}

// The command the whole change is about. Named on its own because the general
// invariant above is satisfied by a command that advertises nothing and
// implements nothing — which is the state db:execute was in when a dump of the
// carrier accounts was archived as a `.json` file that is not JSON.
func TestDbExecuteAnswersInJSON(t *testing.T) {
	def, ok := command.Get("db:execute")
	if !ok {
		t.Fatal("db:execute is not registered")
	}
	if !def.JSONOutput {
		t.Error("db:execute is not marked JSONOutput, so --json is refused instead of answered")
	}
}
