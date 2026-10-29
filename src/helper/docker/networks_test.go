package docker

import (
	"reflect"
	"strings"
	"testing"
)

// Which networks madock may give back when the address pools run out. A wrong
// answer removes somebody else's compose network.
func TestIsMadockNetwork(t *testing.T) {
	cases := map[string]bool{
		"madock_shop-clean":   true,
		"madock_dbg4":         true,
		"madock_":             false,
		"aruntime":            false,
		"someone-elses-stack": false,
		"":                    false,
	}
	for label, want := range cases {
		if got := isMadockNetwork(label); got != want {
			t.Errorf("isMadockNetwork(%q) = %v, want %v", label, got, want)
		}
	}
}

// The error is the last thing docker prints; the progress lines before it are
// what made the message useless when all of them were kept or none were.
func TestLastLinesKeepsTheEnd(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 30; i++ {
		b.WriteString(" Container x Created\n")
	}
	b.WriteString("failed to create network madock_dbg4_default: " + poolsExhausted + "\n")

	got := lastLines(b.String(), 3)
	if !strings.HasSuffix(got, poolsExhausted) {
		t.Fatalf("the error line was lost: %q", got)
	}
	if n := len(strings.Split(got, "\n")); n != 3 {
		t.Errorf("kept %d lines, want 3", n)
	}
	if lastLines("  \n", 5) != "" {
		t.Error("an empty stderr must give no detail")
	}
}

// A clone whose first start was narrowed to db, nginx and php never created
// valkey; compose start then woke the three and said nothing about the fourth.
func TestMissingFromNamesServicesWithoutAContainer(t *testing.T) {
	declared := map[string]bool{"db": true, "nginx": true, "php": true, "valkey": true}
	entries := []ServiceState{
		{Service: "db", State: "running"},
		{Service: "nginx", State: "running"},
		// Stopped is not missing: compose start wakes it.
		{Service: "php", State: "exited"},
		// Present but not declared — the snapshot helper, a removed service.
		{Service: "snapshot", State: "exited"},
	}

	if got := missingFrom(declared, entries); !reflect.DeepEqual(got, []string{"valkey"}) {
		t.Fatalf("missing = %v, want [valkey]", got)
	}
	if got := missingFrom(declared, append(entries, ServiceState{Service: "valkey", State: "created"})); got != nil {
		t.Fatalf("missing = %v, want none", got)
	}
}
