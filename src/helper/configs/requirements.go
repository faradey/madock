package configs

import (
	"regexp"
	"strings"

	"github.com/faradey/madock/v4/src/version"
)

// A project can say which madock it needs: madock/min_version in its own
// .madock/config.xml.
//
// That file is committed, so every developer on a project reads the same one,
// and they do not all run the same madock. A newer madock keeps reading old
// names (see renamed.go), so an old file is never the problem. A new one is: a
// key, a template function or a snippet that an older madock does not know is
// read as absent — often silently. The older binary cannot be taught after the
// fact; what it can do, from the version that carries this check, is refuse to
// work on a project that says it needs more, and say what to update to.
//
// madock never writes the key and never creates the file for it: it is the
// project's statement, written by whoever starts relying on something new.
// config:set refuses it for the same reason — no default declares it.
//
// An edition adds its own key with RegisterVersionRequirement, so one check
// answers for both binaries.

type versionRequirement struct {
	key     string
	edition string
	current string
}

var versionRequirements = []versionRequirement{
	{key: "madock/min_version", edition: "madock", current: version.Version},
}

var plainVersion = regexp.MustCompile(`^v?[0-9]+(\.[0-9]+){0,2}$`)

// RegisterVersionRequirement adds a key a project may set to demand at least
// that version of an edition.
func RegisterVersionRequirement(key, edition, current string) {
	versionRequirements = append(versionRequirements, versionRequirement{key: key, edition: edition, current: current})
}

// UnmetVersionRequirements lists, as sentences, every version the project asks
// for and this binary does not meet. A value that is not a version is unmet
// too: guessing what it meant would be the silent reading this exists to stop.
func UnmetVersionRequirements(conf map[string]string) []string {
	var unmet []string
	for _, req := range versionRequirements {
		wanted := strings.TrimSpace(conf[req.key])
		if wanted == "" {
			continue
		}
		if !plainVersion.MatchString(wanted) {
			unmet = append(unmet, "This project's "+req.key+" is \""+wanted+"\", which is not a version number")
			continue
		}
		if CompareVersions(strings.TrimPrefix(req.current, "v"), strings.TrimPrefix(wanted, "v")) < 0 {
			unmet = append(unmet, "This project needs "+req.edition+" "+strings.TrimPrefix(wanted, "v")+" or newer ("+req.key+"); this is "+req.edition+" "+req.current)
		}
	}
	return unmet
}
