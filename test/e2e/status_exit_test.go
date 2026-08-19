//go:build e2e

package e2e

import (
	"testing"
	"time"
)

// TestStatusAnswersZeroWhenNothingRuns pins a decision rather than a fix.
//
// The question was whether `status` should end non-zero when a project has no
// services running, since `start` and `setup` already do. It should not, and the
// rule is worth stating: zero means the question was answered. "Nothing is
// running" is a true answer to "what is running" — a script that reads it as a
// failure cannot tell a stopped project from a broken one, and the exit code is
// the only thing left to say "I could not look", which is what it says when
// docker cannot be asked.
//
// p.run fails the test on a non-zero exit, so the call itself is the assertion;
// the text check is there so a future change to the message does not quietly
// leave this passing against nothing.
func TestStatusAnswersZeroWhenNothingRuns(t *testing.T) {
	p := newProject(t, "e2estatusidle")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2estatusidle.test",
	)

	// Never started: there are no containers at all, which is the emptiest the
	// answer gets.
	requireContains(t, p.run(2*time.Minute, "status"), "No services found",
		"status on a project that was set up and never started")
}
