package logs

import (
	"errors"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/logrotate"
)

// withRotate replaces the thing that talks to docker, so these tests measure
// the decisions rather than a daemon.
func withRotate(t *testing.T, fn func(container string, plan logrotate.Plan) (string, error)) *[]string {
	t.Helper()

	var seen []string
	previous := logrotate.Rotate
	logrotate.Rotate = func(container string, plan logrotate.Plan) (string, error) {
		seen = append(seen, container)

		return fn(container, plan)
	}
	t.Cleanup(func() { logrotate.Rotate = previous })

	return &seen
}

func conf(extra map[string]string) map[string]string {
	base := map[string]string{
		"logs/persist/enabled":  "true",
		"logs/persist/keep":     "7",
		"logs/persist/max_size": "50M",
		"container_name_prefix": "madock_",
	}
	for key, value := range extra {
		base[key] = value
	}

	return base
}

// Off means nothing is asked of docker at all. A rotation that ran anyway would
// create files in a project that decided it did not want them.
func TestRotationDoesNothingWhenPersistenceIsOff(t *testing.T) {
	seen := withRotate(t, func(string, logrotate.Plan) (string, error) { return "", nil })

	RotateProject("someproject", conf(map[string]string{"logs/persist/enabled": "false"}))

	if len(*seen) != 0 {
		t.Errorf("rotation ran for a project with persistence off: %v", *seen)
	}
}

// The numbers a project configured have to arrive at the plan, or rotation
// happens at a size and a count nobody chose.
func TestTheConfiguredNumbersReachThePlan(t *testing.T) {
	var got logrotate.Plan
	withRotate(t, func(_ string, plan logrotate.Plan) (string, error) {
		got = plan

		return "", nil
	})

	RotateProject("someproject", conf(map[string]string{"logs/persist/keep": "3", "logs/persist/max_size": "10M"}))

	if got.Keep != 3 || got.MaxSize != 10<<20 {
		t.Errorf("plan is keep=%d max=%d, expected 3 and 10M", got.Keep, got.MaxSize)
	}
}

// A container that is not there is the ordinary case — a project without a
// database, a service switched off — and it must not produce a warning, or the
// warning that matters is lost among them.
func TestAnAbsentContainerIsNotAComplaint(t *testing.T) {
	withRotate(t, func(container string, _ logrotate.Plan) (string, error) {
		return "Error response from daemon: No such container: " + container, errors.New("exit status 1")
	})

	if lines := RotateProject("someproject", conf(nil)); len(lines) != 0 {
		t.Errorf("an absent container was reported as work done: %v", lines)
	}
}

// Both services that mount the volume are asked, and the names are the ones
// docker knows.
func TestEveryServiceThatKeepsFilesIsRotated(t *testing.T) {
	seen := withRotate(t, func(string, logrotate.Plan) (string, error) { return "", nil })

	RotateProject("someproject", conf(nil))

	joined := strings.Join(*seen, " ")
	for _, want := range []string{"madock_someproject-nginx-1", "madock_someproject-db-1"} {
		if !strings.Contains(joined, want) {
			t.Errorf("%s was never rotated, only: %v", want, *seen)
		}
	}
}

// A size nobody can read stops the rotation rather than inventing a threshold:
// a guessed default here means "rotate later than you asked", which is found as
// a full disk.
func TestAnUnreadableSizeStopsTheRotation(t *testing.T) {
	seen := withRotate(t, func(string, logrotate.Plan) (string, error) { return "", nil })

	RotateProject("someproject", conf(map[string]string{"logs/persist/max_size": "enormous"}))

	if len(*seen) != 0 {
		t.Errorf("rotation ran with an unreadable threshold: %v", *seen)
	}
}
