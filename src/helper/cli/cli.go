package cli

import (
	"regexp"
	"strings"

	"github.com/faradey/madock/v4/src/helper/cli/shell"
)

// assignmentRe matches an argument that is a shell variable assignment, so the
// name can be left bare while the value is quoted. Quoting the whole word would
// stop it being an assignment at all: bash only recognises one when the `=` is
// unquoted, so 'FOO=bar baz' is a command named FOO=bar baz.
var assignmentRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

// NormalizeCliCommand prepares argv for a command line that will be handed to
// `bash -c` inside a container.
//
// Every passthrough command in madock ends up here — cli, magento, composer,
// node, n98, shopware and the rest — and each of them joins the result into one
// string. That join is the whole problem this function has to solve: argv comes
// in already split by the user's shell, with the quoting removed, and putting it
// back together means the container's shell splits it a second time, by its own
// rules rather than the ones the user typed.
//
// The result was that any argument carrying shell syntax broke or, worse,
// changed meaning:
//
//	madock cli node -e "console.log(process.version)"
//	bash: -c: line 1: syntax error near unexpected token `('
//
// The parentheses never reached node — they reached bash. The old code quoted an
// argument only when it contained a space or an `=`, which covers neither
// brackets, nor `|`, `;`, `&`, `*`, `$`, backticks, nor a newline. It also
// wrapped values in double quotes, which keep `$` and backticks live, and it
// stripped quote characters off the ends of values outright.
//
// Now every argument is quoted for the shell, so it arrives as the one word it
// was. Two behaviours are kept deliberately:
//
//   - A single argument is passed through untouched. `madock cli "ls | grep x"`
//     is the established way to run a pipeline, and there is no way to both quote
//     that string and keep it a pipeline. One argument means "this is a script";
//     several mean "this is argv".
//   - NAME=value keeps its shape, with only the value quoted, so environment
//     prefixes such as `madock cli APP_ENV=dev php bin/console` still assign.
//
// Arguments are no longer trimmed of surrounding whitespace: a function whose
// job is to deliver an argument unchanged has no business editing it.
func NormalizeCliCommand(arguments []string) []string {
	if len(arguments) <= 1 {
		return arguments
	}

	args := make([]string, len(arguments))
	for i, val := range arguments {
		if m := assignmentRe.FindStringSubmatch(val); m != nil {
			args[i] = m[1] + "=" + shell.Quote(m[2])
			continue
		}
		args[i] = shell.Quote(val)
	}

	return args
}

func NormalizeCliCommandWithJoin(arguments []string) string {
	return strings.Join(NormalizeCliCommand(arguments), " ")
}
