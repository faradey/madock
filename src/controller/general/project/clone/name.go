package clone

import "strings"

// nameRefusal says why a clone name cannot be used, or "" when it can.
//
// The name becomes projects/<name>/ and a directory beside the source, and a
// clone that fails removes both. An empty name would point that removal at
// projects/ itself, every project of the installation; a slash or a `..` at
// somewhere outside it. go-arg's `required` only asks for the flag, so
// `--name=` arrives here as "".
func nameRefusal(name string) string {
	switch {
	case strings.TrimSpace(name) == "":
		return "The project folder name cannot be empty"
	case strings.Contains(name, ".") || strings.Contains(name, " "):
		return "The project folder name cannot contain a period or space"
	case strings.ContainsAny(name, `/\`):
		return "The project folder name cannot contain a slash"
	}
	return ""
}
