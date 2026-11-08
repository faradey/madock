package docker

import (
	"fmt"
	"testing"
	"time"
)

// What is guarded here is a report, not a container: `docker compose ps -a`
// lists by project label, and a label outlives the service it was put on. The
// old code called every stopped container a service that failed to start, so a
// project with `nginx/enabled=false` was told on every single start that its
// nginx had exited — a service `status` does not list, because status reads the
// compose file and the warning read docker.
//
// The second half matters as much: when the declared set cannot be established
// nothing may be called a leftover. A check that cannot tell must fall back to
// what it did before rather than invent a category.

func states(pairs ...[2]string) []ServiceState {
	out := make([]ServiceState, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, ServiceState{Service: p[0], State: p[1]})
	}
	return out
}

func names(entries []ServiceState) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Service)
	}
	return out
}

func equal(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestStoppedContainerOfADeclaredServiceIsAFailure(t *testing.T) {
	dead, orphans := classifyStates(
		states([2]string{"db", "running"}, [2]string{"php", "exited"}),
		map[string]bool{"db": true, "php": true},
	)
	if !equal(names(dead), []string{"php"}) {
		t.Errorf("dead = %v, want [php]", names(dead))
	}
	if len(orphans) != 0 {
		t.Errorf("nothing here is a leftover, got %v", names(orphans))
	}
}

func TestStoppedContainerOfAGoneServiceIsALeftover(t *testing.T) {
	dead, orphans := classifyStates(
		states([2]string{"db", "running"}, [2]string{"nodejs", "running"}, [2]string{"nginx", "exited"}),
		map[string]bool{"db": true, "nodejs": true},
	)
	if len(dead) != 0 {
		t.Errorf("a service the stack does not declare cannot have failed to start: %v", names(dead))
	}
	if !equal(names(orphans), []string{"nginx"}) {
		t.Errorf("orphans = %v, want [nginx]", names(orphans))
	}
}

func TestBothAtOnceAreReportedSeparately(t *testing.T) {
	dead, orphans := classifyStates(
		states([2]string{"php", "exited"}, [2]string{"nginx", "exited"}),
		map[string]bool{"php": true},
	)
	if !equal(names(dead), []string{"php"}) {
		t.Errorf("dead = %v, want [php]", names(dead))
	}
	if !equal(names(orphans), []string{"nginx"}) {
		t.Errorf("orphans = %v, want [nginx]", names(orphans))
	}
}

// The fallback. nil means the question could not be answered — docker silent,
// compose file unreadable — and then the report is exactly what it was before
// this change.
func TestUnknownDeclaredSetCallsNothingALeftover(t *testing.T) {
	dead, orphans := classifyStates(
		states([2]string{"nginx", "exited"}, [2]string{"php", "exited"}),
		nil,
	)
	if !equal(names(dead), []string{"nginx", "php"}) {
		t.Errorf("dead = %v, want both", names(dead))
	}
	if len(orphans) != 0 {
		t.Errorf("nothing may be called a leftover without knowing the stack: %v", names(orphans))
	}
}

// By the time this is asked settledStates has given it its chance.
func TestStillRestartingCountsAsDead(t *testing.T) {
	dead, orphans := classifyStates(
		states([2]string{"php", "restarting"}),
		map[string]bool{"php": true},
	)
	if len(dead) != 1 || len(orphans) != 0 {
		t.Errorf("a service still restarting is not up: dead=%v orphans=%v", names(dead), names(orphans))
	}
}

// withStates scripts the samples settledStates takes. Restart counts stay at zero
// unless withCounts scripts them too, so nothing here asks a real docker.
func withStates(t *testing.T, grace time.Duration, answers ...[]ServiceState) *int {
	t.Helper()
	prevStates, prevCounts, prevGrace, prevPoll := serviceStatesNow, restartCountsNow, restartGrace, restartPoll
	t.Cleanup(func() {
		serviceStatesNow, restartCountsNow, restartGrace, restartPoll = prevStates, prevCounts, prevGrace, prevPoll
	})

	asked := 0
	serviceStatesNow = func(string) ([]ServiceState, error) {
		answer := answers[min(asked, len(answers)-1)]
		asked++
		return answer, nil
	}
	withCounts(0)
	restartGrace, restartPoll = grace, time.Millisecond
	return &asked
}

// withCounts scripts docker's RestartCount for php, one value per sample, the
// last repeated. Call it after withStates.
func withCounts(counts ...int) {
	asked := 0
	restartCountsNow = func(names []string) (map[string]int, error) {
		n := counts[min(asked, len(counts)-1)]
		asked++
		out := map[string]int{}
		for _, name := range names {
			out[name] = n
		}
		return out, nil
	}
}

var (
	allUp     = []ServiceState{{Service: "php", Name: "p-php-1", State: "running"}}
	looping   = []ServiceState{{Service: "php", Name: "p-php-1", State: "restarting", ExitCode: 1}}
	longGrace = time.Minute
)

// The loop state cannot see: docker brings the container back after 100 ms, so a
// service that dies within a second is `running` in every sample. Its restart
// count grows all the same. Measured: such an nginx made start exit 0 in 8 of 9.
func TestALoopSeenRunningInEverySampleIsCaught(t *testing.T) {
	withStates(t, 20*time.Millisecond, allUp)
	counter := 0
	restartCountsNow = func(names []string) (map[string]int, error) {
		counter++
		out := map[string]int{}
		for _, name := range names {
			out[name] = counter
		}
		return out, nil
	}
	entries, err := settledStates("p")
	if err != nil {
		t.Fatal(err)
	}
	if !anyRestarting(entries) {
		t.Errorf("a container restarted on every sample was reported up: %+v", entries)
	}
}

// A service that restarted while waiting for a database and then stays up is up.
func TestALoopThatStopsRestartingIsUp(t *testing.T) {
	withStates(t, longGrace, allUp)
	withCounts(0, 1, 1, 1, 1)
	entries, err := settledStates("p")
	if err != nil {
		t.Fatal(err)
	}
	if anyRestarting(entries) {
		t.Errorf("a service whose count stopped growing was reported restarting: %+v", entries)
	}
}

// Counts that cannot be read are not evidence of anything: the decision is the
// state's, as before, and a stack that is up is up.
func TestUnreadableCountsFallBackToState(t *testing.T) {
	asked := withStates(t, longGrace, allUp)
	restartCountsNow = func([]string) (map[string]int, error) {
		return nil, fmt.Errorf("docker did not answer")
	}
	entries, err := settledStates("p")
	if err != nil {
		t.Fatal(err)
	}
	if anyRestarting(entries) || *asked != 2 {
		t.Errorf("asked %d times, reported %+v", *asked, entries)
	}
}

func TestParseRestartCounts(t *testing.T) {
	got := parseRestartCounts([]byte("/p-php-1 3\n/p-nginx-1 0\ngarbage\n/p-db-1 x\n"))
	if len(got) != 2 || got["p-php-1"] != 3 || got["p-nginx-1"] != 0 {
		t.Errorf("got %v", got)
	}
}

// Most projects: nobody restarting, so one confirming sample and no wait. The
// confirm is there because `compose up` returns before a bad config has had time
// to fail; a stack that is up stays up across both samples.
func TestNothingRestartingConfirmsOnceAndDoesNotWait(t *testing.T) {
	asked := withStates(t, longGrace, allUp, allUp)
	entries, err := settledStates("p")
	if err != nil {
		t.Fatal(err)
	}
	if anyRestarting(entries) {
		t.Errorf("reported %+v", entries)
	}
	if *asked != 2 {
		t.Errorf("asked %d times, want 2 (sample + confirm)", *asked)
	}
}

// The race this closes: `compose up` returns while an nginx that rejects its
// vhost is still in its first running moment, so the first sample sees it up and
// start exited 0 over a service already crash-looping. The confirming sample a
// restartPoll later catches it restarting.
func TestAServiceRunningOnlyBecauseItHasNotCrashedYetIsCaught(t *testing.T) {
	withStates(t, 20*time.Millisecond, allUp, looping)
	entries, err := settledStates("p")
	if err != nil {
		t.Fatal(err)
	}
	if !anyRestarting(entries) {
		t.Errorf("a service that crashed right after up was reported up: %+v", entries)
	}
}

func TestAServiceThatComesUpIsNotReportedAsRestarting(t *testing.T) {
	withStates(t, longGrace, looping, looping, allUp)
	entries, err := settledStates("p")
	if err != nil {
		t.Fatal(err)
	}
	if anyRestarting(entries) {
		t.Errorf("reported %+v", entries)
	}
}

func TestAServiceThatKeepsRestartingIsReportedSo(t *testing.T) {
	withStates(t, 20*time.Millisecond, looping)
	entries, err := settledStates("p")
	if err != nil {
		t.Fatal(err)
	}
	if !anyRestarting(entries) {
		t.Errorf("reported %+v", entries)
	}
}

// The moment such a process lives is not it coming up.
func TestOneRunningSampleInALoopIsNotComingUp(t *testing.T) {
	asked := withStates(t, longGrace, looping, allUp, looping, allUp, allUp)
	if _, err := settledStates("p"); err != nil {
		t.Fatal(err)
	}
	if *asked != 5 {
		t.Errorf("asked %d times: one running sample ended the wait", *asked)
	}
}

func TestEverythingRunningReportsNothing(t *testing.T) {
	dead, orphans := classifyStates(
		states([2]string{"db", "running"}, [2]string{"php", "running"}),
		map[string]bool{"db": true, "php": true},
	)
	if len(dead) != 0 || len(orphans) != 0 {
		t.Errorf("a healthy project must produce no lines at all: dead=%v orphans=%v", names(dead), names(orphans))
	}
}
