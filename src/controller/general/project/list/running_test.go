package list

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/configs"
)

// The gap this closes: there was no way to ask which projects are up. The JSON
// carried name, path and state, where state answers a different question
// (`ok` / `missing-source`), so finding out who was eating memory meant walking
// the registry and running `status` in each project.
//
// Measured cost of not having it: a Shopware administration build was failing on
// OOM — Vite wants about 4 GB in an 8 GB machine — and the first question,
// "what else is running", had no command behind it.
func TestWithRunningMarksTheProjectsThatAreUp(t *testing.T) {
	entries := []configs.ProjectEntry{
		{Name: "alpha", Path: "/srv/alpha", State: configs.ProjectOk},
		{Name: "beta", Path: "/srv/beta", State: configs.ProjectOk},
	}

	rows := withRunning(entries, []string{"beta"}, true)

	if rows[0].Running == nil || *rows[0].Running {
		t.Errorf("alpha is not running and should say so, got %v", rows[0].Running)
	}
	if rows[1].Running == nil || !*rows[1].Running {
		t.Errorf("beta is running and should say so, got %v", rows[1].Running)
	}
}

// Each row needs its own bool. Sharing one variable across the loop gives every
// row a pointer to whatever the last iteration decided — a bug that would show
// as "everything is running" or "nothing is", depending on the last project in
// the registry.
func TestWithRunningGivesEachRowItsOwnAnswer(t *testing.T) {
	entries := []configs.ProjectEntry{
		{Name: "up", State: configs.ProjectOk},
		{Name: "down", State: configs.ProjectOk},
	}

	rows := withRunning(entries, []string{"up"}, true)

	if rows[0].Running == rows[1].Running {
		t.Fatal("two rows share one pointer, so they cannot disagree")
	}
	if !*rows[0].Running || *rows[1].Running {
		t.Errorf("got up=%v down=%v, want true and false", *rows[0].Running, *rows[1].Running)
	}
}

// Docker unavailable is not "nothing is running". It is no answer, and the JSON
// has to say null rather than false — a script reading false would act on a fact
// nobody established.
func TestWithRunningSaysNothingWhenDockerCouldNotBeAsked(t *testing.T) {
	entries := []configs.ProjectEntry{{Name: "alpha", State: configs.ProjectOk}}

	rows := withRunning(entries, nil, false)

	if rows[0].Running != nil {
		t.Fatalf("an unanswered question became an answer: %v", *rows[0].Running)
	}

	out, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("marshalling the rows: %v", err)
	}
	if !strings.Contains(string(out), `"running":null`) {
		t.Errorf("unknown must serialise as null, got %s", out)
	}
}
