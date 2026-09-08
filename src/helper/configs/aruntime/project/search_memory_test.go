package project

import (
	"strings"
	"testing"
)

// The case that was measured: a 768m limit under a 1 GB heap. The container was
// killed during startup and the only visible sign was the service missing from
// `madock status`.
func TestAHeapLargerThanTheLimitIsRefused(t *testing.T) {
	complaint := checkSearchMemory("opensearch", "1g", "768m")
	if complaint == "" {
		t.Fatal("a 1g heap in a 768m container was accepted — that container is killed on start")
	}
	if !strings.Contains(complaint, "search/opensearch/heap") ||
		!strings.Contains(complaint, "search/opensearch/memory_limit") {
		t.Errorf("the complaint does not name both settings:\n%s", complaint)
	}
}

// Equal is refused too: the JVM is not the only thing in the container, so a
// heap exactly the size of the limit has nowhere to live either.
func TestAHeapEqualToTheLimitIsRefused(t *testing.T) {
	if checkSearchMemory("opensearch", "768m", "768m") == "" {
		t.Error("a heap equal to the limit was accepted")
	}
}

// The shipped defaults must pass, or every project stops rendering.
func TestTheShippedDefaultsFit(t *testing.T) {
	if complaint := checkSearchMemory("opensearch", "1g", "2512m"); complaint != "" {
		t.Errorf("the opensearch defaults are refused by their own check:\n%s", complaint)
	}
	if complaint := checkSearchMemory("elasticsearch", "800m", "2512m"); complaint != "" {
		t.Errorf("the elasticsearch defaults are refused by their own check:\n%s", complaint)
	}
}

// Units are read as people write them, across both settings.
func TestUnitsAreComparedNotStrings(t *testing.T) {
	if checkSearchMemory("opensearch", "256m", "1G") != "" {
		t.Error("256m inside 1G was refused")
	}
	if checkSearchMemory("opensearch", "2G", "1024MB") == "" {
		t.Error("2G inside 1024MB was accepted")
	}
}

// A value neither this code nor a person can read is docker's to complain
// about. Guessing here would refuse a form docker accepts, and this check has
// no business being the thing that stops a project from starting.
func TestAnUnreadableValueIsNotThisChecksBusiness(t *testing.T) {
	if checkSearchMemory("opensearch", "big", "768m") != "" {
		t.Error("an unparseable heap produced a complaint")
	}
	if checkSearchMemory("opensearch", "", "") != "" {
		t.Error("empty settings produced a complaint")
	}
}
