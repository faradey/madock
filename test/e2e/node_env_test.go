//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestNodeContainerDoesNotSetNodeEnv pins the one thing the rendered file cannot
// prove: what the container actually ends up with.
//
// madock used to put NODE_ENV in the node container's environment, because the
// entrypoint needs to know whether to start `dev` or `start` when no
// nodejs/script is configured. That is madock's own decision under a name every
// tool in the container obeys — and `next build` obeys it instead of setting
// `production` itself, so a production bundle is built in development mode and
// React's internals disagree with themselves: prerendering `/_global-error`
// dies on `useContext` with a null dispatcher.
//
// The cost was in the finding, not the fix. Next prints "You are using a
// non-standard NODE_ENV value" as the first line of every build, and that line
// went past about ten times — a warning printed always is indistinguishable
// from noise until it is the cause. React, Node, Next, styled-components and
// the application's own layout were all ruled out first, and an application of
// two files failed the same way.
//
// The decision now travels as MADOCK_NODE_ENV. A project that wants NODE_ENV
// sets it itself, in a package.json script, where it applies to one command
// rather than to everything in the container.
func TestNodeContainerDoesNotSetNodeEnv(t *testing.T) {
	p := newProject(t, "e2enodeenv")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=nodejs",
		"--hosts=e2enodeenv.test",
	)
	p.run(20*time.Minute, "start")

	// printenv exits non-zero when the variable is unset, which is the passing
	// case here — so this asks rather than asserts through the exit code.
	out, err := p.tryRun(2*time.Minute, "cli", "--service", "nodejs", "printenv", "NODE_ENV")
	if err == nil && strings.TrimSpace(lastLine(out)) != "" {
		t.Errorf("the node container carries NODE_ENV=%s, so `next build` will build a production bundle in development mode:\n%s",
			strings.TrimSpace(lastLine(out)), out)
	}

	// And the decision madock does need is still there, under its own name.
	own := p.run(2*time.Minute, "cli", "--service", "nodejs", "printenv", "MADOCK_NODE_ENV")
	if !strings.Contains(own, "development") {
		t.Errorf("MADOCK_NODE_ENV is not set, so the entrypoint cannot tell dev from start:\n%s", own)
	}
}

// lastLine returns the last non-empty line, which is where a command's answer
// is when madock has printed anything of its own first.
func lastLine(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return lines[i]
		}
	}

	return ""
}
