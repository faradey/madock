package docker

import (
	"errors"
	"strings"
	"testing"
)

// Removing a stopped project used to print "could not reclaim … is not
// running" every time, although the throwaway container then did the work.
func TestReclaimIsQuietWhenTheThrowawayWorks(t *testing.T) {
	stopped := func() ([]byte, error) { return []byte("container is not running"), errors.New("exit status 1") }
	worked := func() ([]byte, error) { return nil, nil }
	if lines := reclaim(stopped, worked, "/var/www/html through p-php-1", "/home/u/p"); len(lines) != 0 {
		t.Errorf("a reclaim that worked printed %q", lines)
	}
}

func TestReclaimReportsBothReasonsWhenBothFail(t *testing.T) {
	stopped := func() ([]byte, error) { return []byte("container is not running"), errors.New("exit status 1") }
	noImage := func() ([]byte, error) { return []byte("pull access denied"), errors.New("exit status 125") }
	lines := reclaim(stopped, noImage, "/var/www/html through p-php-1", "/home/u/p")
	if len(lines) != 2 || !strings.Contains(lines[0], "not running") || !strings.Contains(lines[1], "pull access denied") {
		t.Errorf("both failures should be reported with their reasons, got %q", lines)
	}
}

func TestReclaimDoesNotRunTheThrowawayWhenTheContainerWorks(t *testing.T) {
	ran := false
	throwaway := func() ([]byte, error) { ran = true; return nil, nil }
	if lines := reclaim(func() ([]byte, error) { return nil, nil }, throwaway, "a", "b"); len(lines) != 0 || ran {
		t.Errorf("lines %q, throwaway ran: %v", lines, ran)
	}
}
