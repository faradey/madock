package tmpl

import (
	"regexp"
	"strings"
)

// shellSyntax is what makes a worker command a shell script rather than one
// program: a separator, a pipe or redirection, a substitution, or a shell
// builtin at the front.
var shellSyntax = regexp.MustCompile("[;&|<>`\\n]|\\$\\(|^\\s*(set|cd|export|source|\\.|exec)\\s")

// workerCommand is what a worker's `sh -c` runs.
//
// A single program is run with `exec`, so it becomes the container's process
// and gets the signals compose sends. A command with shell syntax in it cannot
// be: `exec a && b` replaces the shell with a and never runs b, and
// `exec set -a; . ./.env; php artisan queue:work` fails on `exec set`. Found by
// the Indemorio engine (IND-11), whose workers load their .env first. Those
// are run by the shell as written.
//
// The result goes inside a double-quoted YAML string that compose then reads,
// so it is escaped for both: a quote or a backslash would end the string — a
// worker that sourced "$VAR" broke the whole compose file — and a $ would be
// substituted by compose from the host instead of by the shell in the
// container, where the worker's variables are.
func workerCommand(command string) string {
	if !shellSyntax.MatchString(command) {
		command = "exec " + command
	}
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "$", "$$").Replace(command)
}
