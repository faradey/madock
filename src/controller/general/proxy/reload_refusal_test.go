package proxy

import (
	"strings"
	"testing"
)

func TestReloadRefusalSaysWhichFailureItIs(t *testing.T) {
	message, todo := reloadRefusal(true)
	if strings.Contains(message, "not running") || strings.Contains(todo, "proxy:start") || strings.Contains(todo, "rebuild") {
		t.Errorf("running: %q, %q", message, todo)
	}

	message, todo = reloadRefusal(false)
	if !strings.Contains(message, "not running") || !strings.Contains(todo, "proxy:start") {
		t.Errorf("not running: %q, %q", message, todo)
	}
}
