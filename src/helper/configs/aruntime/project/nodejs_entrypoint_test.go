package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// These run the entrypoint the image would run, against fake `node` and `npm`.
//
// The two defects they exist for were both invisible to a rendering test — the
// file looked right and the container did the wrong thing:
//
//   - `--inspect-brk` in the container's environment was inherited by the
//     entrypoint's own `node -e` helper, which stopped before its first line and
//     waited for a debugger that was never coming. Its stderr goes to /dev/null,
//     so the container hung with nothing said.
//   - the inspector port was taken by the package manager. yarn, npm and pnpm
//     are Node programs; with NODE_OPTIONS set they open the port first and keep
//     it, the dev server they spawn fails to bind and runs without a debugger,
//     and the IDE attaches to the package manager.
//
// So the assertions are about which process ends up with the inspector, which is
// a question only running it can answer.
func TestNodejsEntrypointDebugging(t *testing.T) {
	t.Run("a configured command is what gets the inspector", func(t *testing.T) {
		out := runEntrypoint(t, map[string]string{
			"nodejs/enabled":       "true",
			"nodejs/debug/enabled": "true",
			"nodejs/script":        "node server.js",
			"nodejs/script_type":   "command",
		}, map[string]string{"MADOCK_DEBUG_PORT": "9229"})

		if !strings.Contains(out, "debugger listening on 0.0.0.0:9229") {
			t.Errorf("the debugger was not announced:\n%s", out)
		}
		// The fake node prints what it was given. This is the assertion that
		// matters: the application process carries the inspector.
		if !strings.Contains(out, "node saw NODE_OPTIONS=--inspect=0.0.0.0:9229") {
			t.Errorf("the application did not start under the debugger:\n%s", out)
		}
	})

	t.Run("break asks for the flag that waits", func(t *testing.T) {
		out := runEntrypoint(t, map[string]string{
			"nodejs/enabled":       "true",
			"nodejs/debug/enabled": "true",
			"nodejs/debug/break":   "true",
			"nodejs/script":        "node server.js",
			"nodejs/script_type":   "command",
		}, map[string]string{"MADOCK_DEBUG_PORT": "9229", "MADOCK_DEBUG_BREAK": "true"})

		if !strings.Contains(out, "node saw NODE_OPTIONS=--inspect-brk=0.0.0.0:9229") {
			t.Errorf("nodejs/debug/break did not reach the application:\n%s", out)
		}
		if !strings.Contains(out, "nothing runs until a debugger attaches") {
			t.Errorf("a container that will sit and wait did not say so:\n%s", out)
		}
	})

	t.Run("a package script says why it cannot be debugged, and still starts", func(t *testing.T) {
		out := runEntrypoint(t, map[string]string{
			"nodejs/enabled":       "true",
			"nodejs/debug/enabled": "true",
			"nodejs/script":        "dev",
			"nodejs/script_type":   "package",
		}, map[string]string{"MADOCK_DEBUG_PORT": "9229"})

		if !strings.Contains(out, "would take the inspector port") {
			t.Errorf("the package-manager case was not explained:\n%s", out)
		}
		// The inspector must not be handed to the package manager: it would hold
		// the port and the application would run without a debugger.
		if strings.Contains(out, "npm saw NODE_OPTIONS=--inspect") {
			t.Errorf("the package manager was given the inspector:\n%s", out)
		}
		// And the project still runs — the person asked for a debugger, not for
		// the container to stop.
		if !strings.Contains(out, "npm ran: run dev") {
			t.Errorf("the project did not start at all:\n%s", out)
		}
	})

	t.Run("the entrypoint's own node helper never inherits the inspector", func(t *testing.T) {
		// script_type=auto makes the entrypoint ask package.json whether "dev"
		// is a script, which it does by running node. With --inspect-brk in the
		// environment that helper used to stop and never return.
		out := runEntrypoint(t, map[string]string{
			"nodejs/enabled":       "true",
			"nodejs/debug/enabled": "true",
			"nodejs/debug/break":   "true",
			"nodejs/script":        "dev",
			"nodejs/script_type":   "auto",
		}, map[string]string{
			"MADOCK_DEBUG_PORT": "9229", "MADOCK_DEBUG_BREAK": "true",
			// Belt and braces: even a NODE_OPTIONS the person set themselves
			// must not reach the helper.
			"NODE_OPTIONS": "--inspect-brk=0.0.0.0:9229",
		})

		if strings.Contains(out, "helper saw NODE_OPTIONS=--inspect") {
			t.Errorf("the entrypoint's own node call inherited the inspector and would hang:\n%s", out)
		}
		if !strings.Contains(out, "npm ran: run dev") {
			t.Errorf("the entrypoint did not get as far as starting anything:\n%s", out)
		}
	})
}

// runEntrypoint renders the nodejs image for a project, lifts the entrypoint out
// of it, and runs that script against fake tools in a fake project.
func runEntrypoint(t *testing.T, overrides map[string]string, env map[string]string) string {
	t.Helper()

	projectName := "nodeentry"
	// A platform with no nodejs image of its own, so the node service is built
	// from the general template — the one carrying the smart entrypoint. magento2
	// ships its own and would not exercise this at all.
	overrides["platform"] = "custom"
	installation := testenv.SetupWith(t, projectName, "nodeentry.test", overrides)

	MakeConf(projectName)

	rendered := testenv.Collect(t, filepath.Join(installation.ExecDir, "aruntime", "projects", projectName), installation)

	var dockerfile string
	for name, body := range rendered {
		if strings.HasSuffix(name, "ctx/nodejs.Dockerfile") {
			dockerfile = body
		}
	}
	if dockerfile == "" {
		t.Fatal("MakeConf produced no nodejs.Dockerfile")
	}

	script := entrypointBody(t, dockerfile)

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "madock-entrypoint")
	writeExecutable(t, scriptPath, script)

	// A project just complete enough for the entrypoint to stop waiting: it
	// blocks until package.json exists and until an installed-dependencies
	// marker appears, both of which are separate defences worth keeping.
	project := filepath.Join(dir, "project")
	writeFile(t, filepath.Join(project, "package.json"), `{"scripts":{"dev":"vite"}}`)
	writeFile(t, filepath.Join(project, "node_modules", ".package-lock.json"), "{}")

	bin := filepath.Join(dir, "bin")
	// The helper call is `node -e ...`; anything else is the application. The
	// two are told apart so a test can say which of them got the inspector.
	writeExecutable(t, filepath.Join(bin, "node"), `#!/bin/sh
if [ "$1" = "-e" ]; then
  echo "helper saw NODE_OPTIONS=${NODE_OPTIONS:-<unset>}"
  exit 0
fi
echo "node saw NODE_OPTIONS=${NODE_OPTIONS:-<unset>}"
echo "node ran: $*"
`)
	writeExecutable(t, filepath.Join(bin, "npm"), `#!/bin/sh
echo "npm saw NODE_OPTIONS=${NODE_OPTIONS:-<unset>}"
echo "npm ran: $*"
`)

	cmd := exec.Command("/bin/sh", scriptPath)
	cmd.Dir = project
	cmd.Env = append(os.Environ(),
		"PATH="+bin+":"+os.Getenv("PATH"),
		"WORKDIR="+project,
	)
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the entrypoint failed: %v\n%s", err, out)
	}

	return string(out)
}

// entrypointBody lifts the script out of the `cat > … <<'MADOCK_EOF'` block that
// writes it into the image.
func entrypointBody(t *testing.T, dockerfile string) string {
	t.Helper()

	const marker = "MADOCK_EOF"

	_, after, found := strings.Cut(dockerfile, marker)
	if !found {
		t.Fatal("the rendered Dockerfile has no entrypoint heredoc")
	}
	body, _, found := strings.Cut(after, marker)
	if !found {
		t.Fatal("the entrypoint heredoc is never closed")
	}

	return strings.TrimPrefix(body, "\n")
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeExecutable(t *testing.T, path, body string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}
