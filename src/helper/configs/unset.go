package configs

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/paths"
)

// A machine can now remove a setting a project ships, which no layer could do
// before.
//
// The layers fill each other's gaps — `ConfigMapping` writes only keys that are
// missing — so the committed `.madock/config.xml` wins every key it declares and
// nothing on the machine can take one away. That is fine until a project is
// deployed somewhere it was not written for: the demo server inherited the
// production hostnames, and the only ways out were editing a file that belongs
// to the repository or maintaining a fork of it.
//
// The declaration is a block naming paths, mirrored as a tree so that one shape
// serves both the parser and the writer:
//
//	<unset>
//	    <nginx><hosts><www>1</www></hosts></nginx>
//	</unset>
//
// which flattens to `unset/nginx/hosts/www` = "1". The value is a marker and is
// never read: under `<unset>` there is nothing else a value could mean. A path
// removes the key **and everything beneath it**, because the case this exists
// for — one host — is several keys (`.../name`, `.../code`) and naming them one
// by one is how one gets left behind.

// unsetPrefix is where a declaration lives in the flattened map.
const unsetPrefix = "unset/"

// protectedFromUnset are the keys a project cannot render without.
//
// Removing one does not produce an error, it produces a stack whose templates
// interpolate an empty string — a vhost with no root, an image with no version.
// The refusal names the key, because a silently ignored line in a config file is
// worse than either outcome.
//
// Deliberately short. It is the list of keys whose absence breaks rendering, not
// a list of keys somebody would rather keep: a longer list is a machine that
// cannot fix itself, which is what this feature exists to allow.
var protectedFromUnset = map[string]bool{
	"path":        true,
	"platform":    true,
	"php/version": true,
}

// unsetKeys returns the paths a layer asks to have removed from the layers
// above it, and reports the protected ones instead of obeying them.
func unsetKeys(layer map[string]string) []string {
	var keys []string
	for key := range layer {
		if !strings.HasPrefix(key, unsetPrefix) {
			continue
		}
		path := strings.TrimPrefix(key, unsetPrefix)
		if path == "" {
			continue
		}
		if protectedFromUnset[path] {
			warnOnce("\"" + path + "\" cannot be unset: nothing renders without it. The line was ignored.")
			continue
		}
		keys = append(keys, path)
	}

	// Sorted so that two runs of the same configuration report the same thing;
	// map order would otherwise shuffle the lines `config:list` prints.
	sort.Strings(keys)

	return keys
}

// stripUnset takes the declarations out of a layer, so they never reach a
// template as if they were settings of their own.
func stripUnset(layer map[string]string) {
	for key := range layer {
		if strings.HasPrefix(key, unsetPrefix) {
			delete(layer, key)
		}
	}
}

// applyUnset removes the named paths, and everything beneath them, from a layer
// that would otherwise win. It returns what it actually removed.
func applyUnset(paths []string, above map[string]string) []string {
	var removed []string
	for _, path := range paths {
		for key := range above {
			if key != path && !strings.HasPrefix(key, path+"/") {
				continue
			}
			delete(above, key)
			removed = append(removed, key)
		}
	}
	sort.Strings(removed)

	return removed
}

// refuseUnsetInTheProjectFile reports a declaration in the committed
// `.madock/config.xml` and removes it.
//
// It is refused for two reasons, and the second is the one that makes it a
// refusal rather than a convention. **It cannot work**: an unset removes keys
// from the layers *above* the file that declares it, and there is no layer above
// the project's own file — it is the strongest. And **it should not work**: that
// file arrives by `git pull` and a deploy, so a line in it takes effect on every
// machine at once without anybody on those machines deciding anything. Removal
// is the one operation where that is worth refusing outright.
//
// The block is dropped and the run continues. Stopping instead would let one
// committed line take every madock command on the machine down, which is a
// larger hole than the one being closed.
func refuseUnsetInTheProjectFile(layer map[string]string) {
	for key := range layer {
		if !strings.HasPrefix(key, unsetPrefix) {
			continue
		}
		warnOnce("<unset> in the project's own .madock/config.xml was ignored: " +
			"it removes keys from the layers above it, and there is no layer above that file. " +
			"Declare it in <install>/projects/<name>/config.xml, which belongs to this machine.")

		break
	}
	stripUnset(layer)
}

// warnedUnset keeps each warning to one line per run. madock reads its
// configuration many times per command, and a warning printed on every read is
// a warning nobody reads.
var warnedUnset sync.Map

func warnOnce(message string) {
	if _, seen := warnedUnset.LoadOrStore(message, true); seen {
		return
	}
	fmtc.WarningLn(message)
}

// DeclareUnset records that this machine wants a key kept out, whatever the
// project's committed file says.
//
// The declaration goes into the machine's own config as `unset/<path>` with a
// marker value, which is the same shape the reader looks for and the same shape
// the writer already knows how to render — so `config:unset --machine` and a
// hand-edited file produce the same file.
func DeclareUnset(projectName, path, activeScope string) error {
	if path == "" {
		return fmt.Errorf("no key given")
	}
	if protectedFromUnset[path] {
		return fmt.Errorf("%q cannot be unset: nothing renders without it", path)
	}

	SetParam(projectName, unsetPrefix+path, "1", activeScope, "")

	return nil
}

// DeclaredUnsets lists what this machine has asked to remove, so `config:list`
// can say it. Without this the effect is invisible: the key is simply absent,
// which looks exactly like a key nobody ever set.
func DeclaredUnsets(projectName string) []string {
	file := paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml"
	if !paths.IsFileExist(file) {
		return nil
	}

	config := ParseXmlFile(file)
	activeScope := "default"
	layer := getConfigByScope(config, activeScope)
	if scope, ok := config["activeScope"]; ok && scope != "" {
		activeScope = scope
		active := getConfigByScope(config, activeScope)
		ConfigMapping(layer, active)
		layer = active
	}

	return unsetKeys(layer)
}

// ProtectedFromUnset is the list a command prints when somebody asks for one of
// them. Exported as a copy: a caller that could edit it could turn the refusal
// off from anywhere.
func ProtectedFromUnset() []string {
	keys := make([]string, 0, len(protectedFromUnset))
	for key := range protectedFromUnset {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}

// GetProjectConfigOnlyMachine is this machine's own layer for one project,
// scope-resolved and without the declarations.
//
// It exists for `config:list --origin`: telling a reader which file a value came
// from means having the layers separately, and everything else in this package
// hands back the merged answer.
func GetProjectConfigOnlyMachine(projectName string) map[string]string {
	file := paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml"
	if !paths.IsFileExist(file) {
		return map[string]string{}
	}

	config := ParseXmlFile(file)
	activeScope := "default"
	layer := getConfigByScope(config, activeScope)
	if scope, ok := config["activeScope"]; ok && scope != "" {
		active := getConfigByScope(config, scope)
		ConfigMapping(layer, active)
		layer = active
	}
	stripUnset(layer)

	return layer
}
