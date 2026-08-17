package configs

import (
	"os"
	"strings"

	"github.com/faradey/madock/v3/src/helper/paths"
)

// DestructiveKey is the one setting that lives outside the scopes, and the
// placement is the whole point.
//
// Everything else in config.xml describes a project: it sits under
// scopes/<scope>/, a project's own file overrides it, and `config:set` writes
// it. That layering is right for what a project is, and wrong for a guard
// against destroying one — a guard a project can switch off is not a guard, and
// `config:set` now writes to the project, so the switch would be reachable by
// the very command it protects against.
//
// getConfigByScope drops every key that is not under scopes/, so a key at the
// top level cannot be reached by a project's config, by a scope, or by
// `config:set`. Changing it means editing the installation's own file, which is
// what "only the person who administers this machine" means in practice.
const DestructiveKey = "allow_destructive_commands"

// AllowsDestructiveCommands reports whether commands that delete a project's
// data may run on this installation.
//
// It is a guard rail and not a security control, and the difference matters
// enough to write down: the file sits beside the binary and is writable by the
// same user, so anyone who can run the command can also edit the file first. It
// stops a mistake, not a person.
//
// What it stops is worth stopping. `project:remove` ends with RemoveAll on the
// working directory, and the release runbook records that running it in the
// installation directory would have taken madock, its repository and every
// other project's configuration with it. `prune` sounds like tidying and is
// `docker compose down` for the current project — with --with-volumes, the data
// volumes and the images too.
func AllowsDestructiveCommands() bool {
	return !strings.EqualFold(strings.TrimSpace(destructiveSetting()), "false")
}

// destructiveSetting resolves the value through the same three layers every
// other setting uses, minus the two that would defeat it: embedded default,
// then the edition's answer, then the installation's own file. No project, no
// scope.
func destructiveSetting() string {
	value := ""

	if len(defaultConfigXML) > 0 {
		value = topLevel(ParseXmlBytes(defaultConfigXML), DestructiveKey, value)
	}

	// An edition may disagree with the community default. madock-pro is the one
	// that runs on servers and says no there.
	if override, ok := defaultOverrides[DestructiveKey]; ok {
		value = override
	}

	// The installation's own file has the last word, because turning this back
	// on must never require a different binary.
	configPath := paths.GetExecDirPath() + "/config.xml"
	if _, err := os.Stat(configPath); err == nil {
		value = topLevel(ParseXmlFile(configPath), DestructiveKey, value)
	}

	return value
}

// topLevel reads a key that sits directly under <config>, returning fallback
// when the file does not carry it.
func topLevel(parsed map[string]string, key, fallback string) string {
	if value, ok := parsed[key]; ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

// DestructiveRefusal is the message a refused command prints.
//
// One text for every such command, because the reader's next question is always
// the same — what do I edit — and answering it differently in each place is how
// half of them end up not answering it at all.
func DestructiveRefusal(commandName string) []string {
	return []string{
		"'" + commandName + "' deletes a project's data, and this installation does not allow that.",
		"",
		"To allow it, set " + DestructiveKey + " to true in:",
		"  " + paths.GetExecDirPath() + "/config.xml",
		"",
		"It sits at the top of the file, outside <scopes>, and no project or",
		"'madock config:set' can change it — which is the point of it.",
	}
}
