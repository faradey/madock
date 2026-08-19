package restart

import (
	"os"
	"testing"

	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
)

// swap replaces the three seams for the duration of one test and restores them.
func swap(t *testing.T, parse func() *arg_struct.ControllerGeneralStart, stop func(), start func(*arg_struct.ControllerGeneralStart)) {
	t.Helper()

	origParse, origStop, origStart := parseArgs, stopContainers, startContainers
	t.Cleanup(func() {
		parseArgs, stopContainers, startContainers = origParse, origStop, origStart
	})

	if parse != nil {
		parseArgs = parse
	}
	stopContainers = stop
	startContainers = start
}

// `madock restart php` is somebody asking for one service, and there is a
// command for that.
//
// go-arg refuses the argument either way — that is the 3.9.16 fix, and it is
// what keeps the environment up. What it cannot do is say what was meant: its
// message reads "too many positional arguments at 'php'", which names neither
// the want nor the command that serves it.
func TestABareServiceNameIsRecognisedAsSuch(t *testing.T) {
	if got := unexpectedPositional([]string{"php"}); got != "php" {
		t.Errorf("unexpectedPositional([php]) = %q, want php", got)
	}
	if got := unexpectedPositional([]string{"--with-chown", "worker-queue"}); got != "worker-queue" {
		t.Errorf("a service name after a flag was not seen: %q", got)
	}
}

// Flags are not service names. Every flag this command takes is a boolean, so
// nothing here is a value belonging to one.
func TestFlagsAreNotMistakenForServiceNames(t *testing.T) {
	for _, argv := range [][]string{
		nil,
		{"--with-chown"},
		{"-c"},
		{"--quiet", "--with-chown"},
	} {
		if got := unexpectedPositional(argv); got != "" {
			t.Errorf("unexpectedPositional(%v) = %q, want none", argv, got)
		}
	}
}

// The command line is read before a single container is stopped.
//
// The order is the fix: go-arg ends the process on an argument the command does
// not take, so parsing after the stop left a production environment down with a
// message that read like nothing had happened.
func TestArgumentsAreReadBeforeContainersStop(t *testing.T) {
	var order []string

	swap(t,
		func() *arg_struct.ControllerGeneralStart {
			order = append(order, "parse")
			return new(arg_struct.ControllerGeneralStart)
		},
		func() { order = append(order, "stop") },
		func(*arg_struct.ControllerGeneralStart) { order = append(order, "start") },
	)

	Execute()

	want := []string{"parse", "stop", "start"}
	if len(order) != len(want) {
		t.Fatalf("got %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("got %v, want %v", order, want)
		}
	}
}

// What was parsed reaches start.
//
// Guarding the half of the fix that is easy to get wrong: attr.Parse answers
// only the first call in a process, so leaving start to parse for itself — the
// obvious way to write this — would hand it a zero struct and drop the flag
// without a word.
func TestParsedFlagsReachStart(t *testing.T) {
	origArgs, origParseState := os.Args, attr.IsParseArgs
	t.Cleanup(func() { os.Args, attr.IsParseArgs = origArgs, origParseState })

	os.Args = []string{"madock", "restart", "--with-chown"}
	attr.IsParseArgs = true

	var got *arg_struct.ControllerGeneralStart
	swap(t, nil, func() {}, func(args *arg_struct.ControllerGeneralStart) { got = args })

	Execute()

	if got == nil {
		t.Fatal("start was never reached")
	}
	if !got.WithChown {
		t.Fatal("--with-chown was parsed by restart but did not reach start")
	}

	// The reason it has to be passed through: a second parse answers nothing.
	if second := attr.Parse(new(arg_struct.ControllerGeneralStart)).(*arg_struct.ControllerGeneralStart); second.WithChown {
		t.Fatal("attr.Parse answered twice — the pass-through is no longer necessary, and this test no longer says why it is there")
	}
}
