package configs

import (
	"sort"
	"strings"

	"github.com/faradey/madock/v4/src/helper/paths"
)

// retiredKeys are settings madock no longer reads, with what decides instead.
//
// search/engine was retired on 2026-03-20 — the engine is whichever of
// search/<engine>/enabled is true — and came back into the defaults when the
// two default files were made one, set to "elasticsearch". Nothing read it, so
// `config:list` reported elasticsearch on a project running OpenSearch, and a
// project that committed `<engine>opensearch</engine>` believed it had chosen
// its engine while the machine's copy of the enabled flags decided it. Found on
// a reference project whose clones came up with the engine setup had left on
// that machine.
var retiredKeys = map[string]string{
	"search/engine": "search/opensearch/enabled or search/elasticsearch/enabled",
}

// RetiredInstead reports whether a key is retired, and what to set instead.
func RetiredInstead(name string) (string, bool) {
	instead, ok := retiredKeys[name]
	return instead, ok
}

// RetiredKey is a retired setting found in a project's own config.
type RetiredKey struct {
	Key     string
	Scope   string
	Instead string
}

// RetiredKeysInProject lists the retired settings the project's committed
// .madock/config.xml still declares, in any scope.
//
// Only that file. The machine's copy under the installation holds them too on
// every project set up before the retirement, and there they are as inert as
// anywhere — but nobody reads that file to learn what the project chose, and
// nobody commits it.
func RetiredKeysInProject(projectPath string) []RetiredKey {
	file := projectPath + "/.madock/config.xml"
	if projectPath == "" || !paths.IsFileExist(file) {
		return nil
	}
	return retiredIn(ParseXmlFile(file))
}

// retiredIn is the decision on a parsed file, testable without one.
func retiredIn(stored map[string]string) []RetiredKey {
	var found []RetiredKey
	for path := range stored {
		rest, ok := strings.CutPrefix(path, "scopes/")
		if !ok {
			continue
		}
		scope, key, ok := strings.Cut(rest, "/")
		if !ok {
			continue
		}
		if instead, retired := retiredKeys[key]; retired {
			found = append(found, RetiredKey{Key: key, Scope: scope, Instead: instead})
		}
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].Scope != found[j].Scope {
			return found[i].Scope < found[j].Scope
		}
		return found[i].Key < found[j].Key
	})
	return found
}
