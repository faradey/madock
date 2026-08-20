package status

import "testing"

// The bug this exists for: `docker compose ps` lists by project, not by file, so
// a container created from an earlier version of the compose file keeps being
// reported after its service is gone from the configuration. `status` presented
// those as ordinary services — a wrong answer rather than a missing one.
//
// It hid a real defect for a day. A project whose config said `db/type: MariaDB`
// generated a compose file with no db service at all, and `status --json` went on
// listing `db` as running; a test written against `status` therefore passed
// against the broken build, and only reading the generated file showed the truth.
func TestMarkOrphansNamesWhatTheConfigurationNoLongerHas(t *testing.T) {
	services := []ServiceStatus{
		{Service: "app", State: "running", Running: true},
		{Service: "db", State: "running", Running: true},
		{Service: "mailpit", State: "exited"},
	}
	// nginx is declared and not running, which is why it is in the set and not in
	// the list: the two sides answer different questions and the check is the
	// difference between them.
	known := map[string]bool{"app": true, "nginx": true}

	got := markOrphans(services, known, true)

	if got[0].Orphan {
		t.Error("app is declared and must not be flagged")
	}
	if !got[1].Orphan {
		t.Error("db is running and is not in the configuration — that is the whole finding")
	}
	// State has nothing to do with it: a stopped leftover is as much a leftover
	// as a running one. The running one is the dangerous half, because it looks
	// exactly like the project working.
	if !got[2].Orphan {
		t.Error("mailpit is not in the configuration; a stopped leftover is still a leftover")
	}
}

// An unanswered question must not become a claim. If compose could not be asked
// which services exist, nothing is flagged — the alternative is a status that
// calls every running container an orphan the first time docker is slow.
func TestMarkOrphansFlagsNothingWhenItCouldNotAsk(t *testing.T) {
	services := []ServiceStatus{
		{Service: "app", State: "running", Running: true},
		{Service: "db", State: "running", Running: true},
	}

	for _, s := range markOrphans(services, nil, false) {
		if s.Orphan {
			t.Errorf("%s was flagged on an answer nobody has: unknown is not the same as absent", s.Service)
		}
	}
}

// A container compose could not name a service for is compose declining to
// answer, not evidence of anything.
func TestMarkOrphansIgnoresAContainerWithNoServiceName(t *testing.T) {
	services := []ServiceStatus{{Name: "stray-container", Service: "", State: "running", Running: true}}

	if markOrphans(services, map[string]bool{"app": true}, true)[0].Orphan {
		t.Error("a container with no service name was flagged; that is silence being read as a fact")
	}
}
