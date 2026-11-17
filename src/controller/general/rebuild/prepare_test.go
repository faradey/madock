package rebuild

import (
	"errors"
	"strings"
	"testing"
)

func recordSteps(t *testing.T, buildErr error) *[]string {
	t.Helper()
	steps := []string{}
	prevRender, prevBuild, prevCached, prevCarry := renderStack, buildWithoutCache, buildImages, carryToPersisted
	renderStack = func(string) { steps = append(steps, "render") }
	buildWithoutCache = func(string) error { steps = append(steps, "build-no-cache"); return buildErr }
	buildImages = func(string) error { steps = append(steps, "build"); return buildErr }
	carryToPersisted = func(string) error { steps = append(steps, "carry"); return nil }
	t.Cleanup(func() {
		renderStack, buildWithoutCache, buildImages, carryToPersisted = prevRender, prevBuild, prevCached, prevCarry
	})
	return &steps
}

// The carry stops the old Redis to copy it. Run before a build that takes
// minutes, it kept the service down for all of them.
func TestTheImagesAreBuiltBeforeRedisIsCarried(t *testing.T) {
	steps := recordSteps(t, nil)

	if err := prepare("p", true); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got := strings.Join(*steps, ","); got != "render,build-no-cache,carry" {
		t.Errorf("steps ran as %s, want render,build-no-cache,carry", got)
	}
}

func TestAFailedBuildCarriesNothing(t *testing.T) {
	for _, noCache := range []bool{true, false} {
		steps := recordSteps(t, errors.New("503"))

		if err := prepare("p", noCache); err == nil {
			t.Fatalf("noCache=%v: a failed build was not reported", noCache)
		}
		if got := *steps; len(got) != 2 || got[0] != "render" {
			t.Errorf("noCache=%v: steps ran as %s — a failed build must leave the service untouched", noCache, strings.Join(got, ","))
		}
	}
}

// Without --no-cache the images are still built before anything is taken
// down. Left to `up --build` after the containers were stopped, the build was
// the outage: about eleven minutes for a live project on 2026-11-17.
func TestWithoutNoCacheTheImagesAreStillBuiltFirst(t *testing.T) {
	steps := recordSteps(t, nil)

	if err := prepare("p", false); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got := strings.Join(*steps, ","); got != "render,build,carry" {
		t.Errorf("steps ran as %s, want render,build,carry", got)
	}
}
