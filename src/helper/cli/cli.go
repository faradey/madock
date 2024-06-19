package cli

import (
	"strings"
)

func NormalizeCliCommand(arguments []string) []string {
	args := arguments
	for i, val := range args {
		val = strings.TrimSpace(val)
		if strings.Contains(val, "=") {
			vals := strings.SplitN(val, "=", 2)
			args[i] = vals[0] + "=\"" + strings.Trim(vals[1], "\"") + "\""
		} else if strings.Contains(val, " ") {
			args[i] = "\"" + strings.Trim(val, "\"") + "\""
		} else {
			args[i] = val
		}
	}
	return args
}

func NormalizeCliCommandWithJoin(arguments []string) string {
	return strings.Join(NormalizeCliCommand(arguments), " ")
}
