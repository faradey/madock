package project

import (
	"errors"
	"strings"
	"testing"
)

// `function "memShareGB" not defined` reads as a broken template and sends
// somebody to edit a file that is correct. Template functions are compiled into
// the binary while templates are files on disk, so it means the binary is older
// than the tree it renders — which the message never said, on a machine where
// the radius is every project at once.
func TestAdviceFor(t *testing.T) {
	advice := adviceFor(errors.New(`template: x.yml:3: function "memShareGB" not defined`))
	if advice == "" {
		t.Fatal("the one error whose message points at the wrong file gets no advice")
	}
	for _, want := range []string{"older than the templates", "Rebuild", "needs no editing"} {
		if !strings.Contains(advice, want) {
			t.Errorf("the advice does not mention %q:\n%s", want, advice)
		}
	}

	// Everything else is left to speak for itself, or the suffix becomes noise
	// attached to errors it does not explain.
	for _, other := range []error{
		nil,
		errors.New("template: x.yml:3: unexpected EOF"),
		errors.New("open /nope: no such file or directory"),
	} {
		if got := adviceFor(other); got != "" {
			t.Errorf("advice was attached to an unrelated error (%v):\n%s", other, got)
		}
	}
}
