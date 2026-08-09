// Package shell builds fragments of commands that are handed to a shell.
//
// Several database commands run through `bash -c` inside a container, because
// what they need cannot be expressed as argv: PGPASSWORD is an environment
// variable, and a dump is a pipeline. Everything placed in such a string comes
// from a project's configuration or from the user's own query, and pasted in
// raw it is read as syntax rather than as data.
package shell

import "strings"

// Quote wraps a value in single quotes for /bin/sh, so that whatever it
// contains arrives as one argument.
//
// A single quote inside the value is the case worth spelling out: it would end
// the string early and hand the remainder to the shell as commands. The
// standard escape closes the quoted run, emits an escaped quote and opens a
// new one — 'it'\''s' is the four tokens the shell then joins back into one.
func Quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
