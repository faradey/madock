package status

import "testing"

// A stopped worker is kept — it is meant to be up, so stopped is a failure —
// and every other stopped service is left out as before: a stopped project
// would otherwise list all of them as exited.
func TestOnlyStoppedWorkersJoinTheRunningServices(t *testing.T) {
	code := 1
	in := []ServiceStatus{
		{Service: "php", State: "running", Running: true},
		{Service: "worker-queue", State: "exited", ExitCode: &code},
		{Service: "db", State: "exited", ExitCode: &code},
	}
	got := withStoppedWorkers(in)
	if len(got) != 2 || got[0].Service != "php" || got[1].Service != "worker-queue" {
		t.Errorf("kept %+v", got)
	}
}
