package helper

import (
	"strings"
)

func NormalizeCliCommand(arguments []string) []string {
	args := arguments
	for i, val := range args {
		args[i] = strings.TrimSpace(val)
	}
	return args
}

func NormalizeCliCommandWithJoin(arguments []string) string {
	return strings.Join(NormalizeCliCommand(arguments), " ")
}
