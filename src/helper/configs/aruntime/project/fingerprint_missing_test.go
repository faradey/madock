package project

import "testing"

// A project whose stack has never been generated has no fingerprint. Answering
// with the hash of the empty set would be a stable, plausible-looking value that
// could then be recorded as "what the containers were built from".
func TestFingerprintOfAnUngeneratedStack(t *testing.T) {
	t.Setenv("MADOCK_EXEC_DIR", t.TempDir())

	if got := Fingerprint("never-generated"); got != "" {
		t.Errorf("Fingerprint = %q for a project with no runtime dir, want empty", got)
	}
}

// RecordApplied must not store that non-answer, or the first real render would
// read as a change and rebuild a project that was only just created.
func TestRecordAppliedSkipsAnUngeneratedStack(t *testing.T) {
	t.Setenv("MADOCK_EXEC_DIR", t.TempDir())

	RecordApplied("never-generated")

	stack(t, "never-generated", map[string]string{"docker-compose.yml": "services: {}"})
	if NeedsRecreate("never-generated") {
		t.Error("the first generated stack read as a change against a recorded non-answer")
	}
}
