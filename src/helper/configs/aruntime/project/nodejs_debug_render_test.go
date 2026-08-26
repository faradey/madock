package project

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/ports"
	"github.com/faradey/madock/v4/src/helper/testenv"
)

// Node's debugger is the first one madock wires up that is not xdebug, and the
// two work in opposite directions: xdebug connects out to the IDE, so PHP
// debugging needed no compose change at all, while node --inspect listens and
// the IDE connects in. That is the whole reason this renders anything.
//
// Gated behind a config key that is off by default, which is the shape that
// renders nothing forever and gets noticed a release later — so both halves are
// asserted, the off one first.
func TestNodejsDebugRendersOnlyWhenAskedFor(t *testing.T) {
	compose := func(t *testing.T, overrides map[string]string) string {
		t.Helper()

		projectName := "nodedbg"
		env := testenv.SetupWith(t, projectName, "nodedbg.test", overrides)

		MakeConf(projectName)

		rendered := testenv.Collect(t, filepath.Join(env.ExecDir, "aruntime", "projects", projectName), env)
		for name, body := range rendered {
			if strings.HasSuffix(name, "docker-compose.yml") {
				return body
			}
		}
		t.Fatal("MakeConf produced no docker-compose.yml")
		return ""
	}

	t.Run("off by default", func(t *testing.T) {
		body := compose(t, map[string]string{"nodejs/enabled": "true"})

		if strings.Contains(body, "9229") {
			t.Errorf("the debug port was published without being asked for:\n%s", body)
		}
		if strings.Contains(body, "MADOCK_DEBUG_PORT") {
			t.Errorf("the runtime was told to debug without being asked for:\n%s", body)
		}
	})

	t.Run("published and switched on when enabled", func(t *testing.T) {
		body := compose(t, map[string]string{
			"nodejs/enabled":       "true",
			"nodejs/debug/enabled": "true",
		})

		// The container side is fixed — inside there is one node process and
		// nothing to collide with. The host side is masked here, because the
		// harness normalises every published port, so what the number is gets
		// asserted against the registry below instead.
		if !strings.Contains(body, `- "<PORT>:9229"`) {
			t.Fatalf("the debug port is not published:\n%s", body)
		}

		// The number comes from the registry and not from the template. That is
		// what keeps two projects debugging at once apart, and it is also why
		// `madock info:ports` needs no change to show it: that command prints
		// every pair the registry holds, whatever allocated them.
		if port := ports.GetRegistry().Get("nodedbg", "nodejs_debug"); port <= 0 {
			t.Errorf("the debug port was not allocated from the registry, so info:ports will not show it: %d", port)
		}

		if !strings.Contains(body, `MADOCK_DEBUG_PORT: "9229"`) {
			t.Errorf("the entrypoint was not told which port to listen on:\n%s", body)
		}

		// **Not** NODE_OPTIONS, and this is the fix rather than a preference.
		// The entrypoint runs `yarn dev` / `npm run dev`, and those are Node
		// programs: they inherit the variable, open the inspector first and keep
		// it, so the dev server they spawn binds nothing and the IDE attaches to
		// the package manager. Only the entrypoint knows what it is about to
		// start, so only it can decide.
		// The key form, not the bare word: the comment above it in the template
		// explains why NODE_OPTIONS is the wrong lever, and naming a thing is
		// not setting it.
		if strings.Contains(body, "NODE_OPTIONS:") {
			t.Errorf("NODE_OPTIONS is set on the container, so the package manager would take the inspector:\n%s", body)
		}

		// Without a break the process must not wait for anybody.
		if strings.Contains(body, "MADOCK_DEBUG_BREAK") {
			t.Errorf("the process was made to wait for an IDE without nodejs/debug/break:\n%s", body)
		}
	})

	// Stopping before the first line is right for debugging a startup problem
	// and wrong for everything else — as a default it would make every debugged
	// container look hung, which is why it is its own switch.
	t.Run("break waits for the IDE", func(t *testing.T) {
		body := compose(t, map[string]string{
			"nodejs/enabled":       "true",
			"nodejs/debug/enabled": "true",
			"nodejs/debug/break":   "true",
		})

		if !strings.Contains(body, `MADOCK_DEBUG_BREAK: "true"`) {
			t.Errorf("nodejs/debug/break did not reach the entrypoint:\n%s", body)
		}
	})
}
