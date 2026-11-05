package restart

import (
	"errors"
	"os"
	"testing"

	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/docker"
)

// stack is what a real project answers: services the config names, services it
// does not (`deployer`, `worker-queue`, `php_without_xdebug`), and a search
// engine whose config key is spelled differently from its service.
var stack = []string{"php", "php_without_xdebug", "nginx", "db", "opensearch", "deployer", "worker-queue"}

// harness replaces the seams for one test and records what reached docker.
type harness struct {
	restarted []string
	reloads   int
	reloadErr error
	states    []docker.ServiceState
	statesErr error
	listErr   error
	exits     []int
}

func newHarness(t *testing.T, argv ...string) *harness {
	t.Helper()

	h := &harness{}

	origList, origRestart, origStates := composeServices, restartServices, serviceStates
	origProject, origExit, origReload := projectName, exit, reloadNginx
	origArgs, origParse := os.Args, attr.IsParseArgs
	t.Cleanup(func() {
		composeServices, restartServices, serviceStates = origList, origRestart, origStates
		projectName, exit, reloadNginx = origProject, origExit, origReload
		os.Args, attr.IsParseArgs = origArgs, origParse
	})

	composeServices = func(string) ([]string, error) {
		if h.listErr != nil {
			return nil, h.listErr
		}
		return stack, nil
	}
	restartServices = func(_ string, services []string) error {
		h.restarted = append(h.restarted, services...)
		return nil
	}
	serviceStates = func(string) ([]docker.ServiceState, error) {
		return h.states, h.statesErr
	}
	reloadNginx = func(string) error {
		h.reloads++
		return h.reloadErr
	}
	projectName = func() string { return "project" }
	exit = func(code int) { h.exits = append(h.exits, code) }

	os.Args = append([]string{"madock", "service:restart"}, argv...)
	attr.IsParseArgs = true

	return h
}

func running(services ...string) []docker.ServiceState {
	states := make([]docker.ServiceState, 0, len(services))
	for _, svc := range services {
		states = append(states, docker.ServiceState{Service: svc, State: "running"})
	}
	return states
}

// The name a person types is not the name compose uses, and three spellings are
// in circulation. All three have to arrive at the same service.
func TestResolveAcceptsEverySpellingInCirculation(t *testing.T) {
	for _, tc := range []struct{ typed, want string }{
		{"php", "php"},
		{"PHP", "php"},
		{"deployer", "deployer"},
		{"worker-queue", "worker-queue"},
		{"search/opensearch", "opensearch"}, // the config key that switches it
	} {
		got, ok := resolve(tc.typed, stack)
		if !ok || got != tc.want {
			t.Errorf("resolve(%q) = %q, %v; want %q, true", tc.typed, got, ok, tc.want)
		}
	}
}

// A name nobody can place is refused rather than guessed at.
func TestResolveRefusesAnUnknownName(t *testing.T) {
	for _, typed := range []string{"", "  ", "phpp", "mysql", "redis"} {
		if got, ok := resolve(typed, stack); ok {
			t.Errorf("resolve(%q) = %q, true; want refusal", typed, got)
		}
	}
}

// Nothing is restarted on a name that does not resolve.
//
// The order matters more than the refusal: with two services named and the
// second misspelled, restarting the first and then refusing leaves the caller
// unable to tell what happened without going and looking.
func TestAnUnknownNameRestartsNothingAtAll(t *testing.T) {
	h := newHarness(t, "php", "redis")

	Execute()

	if len(h.restarted) != 0 {
		t.Fatalf("restarted %v; want nothing", h.restarted)
	}
	if len(h.exits) == 0 || h.exits[0] != 1 {
		t.Fatalf("exits = %v; want the command to fail", h.exits)
	}
}

// The named services reach docker, and only they do.
func TestOnlyTheNamedServicesAreRestarted(t *testing.T) {
	h := newHarness(t, "php", "worker-queue")
	h.states = running("php", "worker-queue", "nginx", "db", "deployer")

	Execute()

	if len(h.restarted) != 2 || h.restarted[0] != "php" || h.restarted[1] != "worker-queue" {
		t.Fatalf("restarted %v; want [php worker-queue]", h.restarted)
	}
	if len(h.exits) != 0 {
		t.Fatalf("exits = %v; want success", h.exits)
	}
}

// A service that does not come back is a failure, even though the restart
// itself returned zero.
//
// `docker compose restart` exits zero once the signals are sent. A worker whose
// command is broken is gone a moment later, and without this check the command
// would report the restart it performed rather than the state it produced —
// which is the same silence the deploy trap is made of.
func TestAServiceThatDoesNotComeBackFailsTheCommand(t *testing.T) {
	h := newHarness(t, "worker-queue")
	h.states = []docker.ServiceState{{Service: "worker-queue", State: "exited", ExitCode: 1}}

	Execute()

	if len(h.restarted) != 1 {
		t.Fatalf("restarted %v; want [worker-queue]", h.restarted)
	}
	if len(h.exits) == 0 || h.exits[len(h.exits)-1] != 1 {
		t.Fatalf("exits = %v; want a failure after a service stayed down", h.exits)
	}
}

// A docker that cannot be asked afterwards is not a failure and not a success.
//
// The restart happened; whether it took is unknown. Reporting that as failure
// would have a recipe roll back a deploy that worked, and reporting it as
// success is the lie this command was written against.
func TestAnUnreadableStateIsNeitherFailureNorSilence(t *testing.T) {
	h := newHarness(t, "php")
	h.statesErr = errors.New("docker daemon is not responding")

	Execute()

	if len(h.restarted) != 1 {
		t.Fatalf("restarted %v; want [php]", h.restarted)
	}
	if len(h.exits) != 0 {
		t.Fatalf("exits = %v; want no failure when the state could not be read", h.exits)
	}
}

// The service list itself failing stops the command before it restarts
// anything, and says which question went unanswered.
func TestAnUnaskableDockerStopsTheCommand(t *testing.T) {
	h := newHarness(t, "php")
	h.listErr = errors.New("docker daemon is not responding")

	Execute()

	if len(h.restarted) != 0 {
		t.Fatalf("restarted %v; want nothing", h.restarted)
	}
	if len(h.exits) == 0 || h.exits[0] != 1 {
		t.Fatalf("exits = %v; want the command to fail", h.exits)
	}
}

// Containers restarted together can swap addresses, and an upstream block holds
// the one nginx read when it loaded its configuration. Without the reload a PHP
// stack keeps talking to an address that now belongs to something else.
func TestARunningNginxIsReloadedAfterARestart(t *testing.T) {
	h := newHarness(t, "php", "worker-queue")
	h.states = running("php", "worker-queue", "nginx", "db")

	Execute()

	if h.reloads != 1 {
		t.Fatalf("nginx reloaded %d times; want once", h.reloads)
	}
}

// An nginx restarted by name has just read its configuration.
func TestAnNginxRestartedByNameIsNotReloaded(t *testing.T) {
	h := newHarness(t, "php", "nginx")
	h.states = running("php", "nginx", "db")

	Execute()

	if h.reloads != 0 {
		t.Fatalf("nginx reloaded %d times; want none", h.reloads)
	}
}

// One that is stopped holds no configuration, and a project may have none at all
// (`nginx/enabled` false).
func TestNoRunningNginxIsNotReloaded(t *testing.T) {
	for name, states := range map[string][]docker.ServiceState{
		"stopped": {{Service: "php", State: "running"}, {Service: "nginx", State: "exited"}},
		"absent":  running("php", "db"),
	} {
		h := newHarness(t, "php")
		h.states = states

		Execute()

		if h.reloads != 0 {
			t.Errorf("%s nginx: reloaded %d times; want none", name, h.reloads)
		}
		if len(h.exits) != 0 {
			t.Errorf("%s nginx: exits = %v; want success", name, h.exits)
		}
	}
}

// The restart happened, so a reload that fails must not read as a failed restart:
// a recipe would roll back a deploy that worked.
func TestAFailedReloadDoesNotFailTheCommand(t *testing.T) {
	h := newHarness(t, "php")
	h.states = running("php", "nginx")
	h.reloadErr = errors.New("nginx -t failed")

	Execute()

	if h.reloads != 1 {
		t.Fatalf("nginx reloaded %d times; want one attempt", h.reloads)
	}
	if len(h.exits) != 0 {
		t.Fatalf("exits = %v; want success", h.exits)
	}
}

// With no name at all the command asks for one instead of restarting the
// project — which is what `restart` is for, and doing it here would be the
// blunt tool wearing the precise name.
func TestNoNameRestartsNothing(t *testing.T) {
	h := newHarness(t)

	Execute()

	if len(h.restarted) != 0 {
		t.Fatalf("restarted %v; want nothing", h.restarted)
	}
	if len(h.exits) == 0 || h.exits[0] != 1 {
		t.Fatalf("exits = %v; want the command to fail", h.exits)
	}
}
