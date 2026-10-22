package rebuild

import (
	"errors"
	"strings"
	"testing"
)

func recordSteps(t *testing.T, buildErr error) *[]string {
	t.Helper()
	steps := []string{}
	prevRender, prevBuild, prevCarry := renderStack, buildWithoutCache, carryToPersisted
	renderStack = func(string) { steps = append(steps, "render") }
	buildWithoutCache = func(string) error { steps = append(steps, "build"); return buildErr }
	carryToPersisted = func(string) error { steps = append(steps, "carry"); return nil }
	t.Cleanup(func() { renderStack, buildWithoutCache, carryToPersisted = prevRender, prevBuild, prevCarry })
	return &steps
}

// The carry stops the old Redis to copy it. Run before a build that takes
// minutes, it kept the service down for all of them.
func TestTheImagesAreBuiltBeforeRedisIsCarried(t *testing.T) {
	steps := recordSteps(t, nil)

	if err := prepare("p", true); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got := strings.Join(*steps, ","); got != "render,build,carry" {
		t.Errorf("steps ran as %s, want render,build,carry", got)
	}
}

func TestAFailedBuildCarriesNothing(t *testing.T) {
	steps := recordSteps(t, errors.New("503"))

	if err := prepare("p", true); err == nil {
		t.Fatal("a failed build was not reported")
	}
	if got := strings.Join(*steps, ","); got != "render,build" {
		t.Errorf("steps ran as %s — a failed build must leave the service untouched", got)
	}
}

func TestWithoutNoCacheOnlyTheCarryRuns(t *testing.T) {
	steps := recordSteps(t, nil)

	if err := prepare("p", false); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got := strings.Join(*steps, ","); got != "carry" {
		t.Errorf("steps ran as %s, want carry", got)
	}
}
