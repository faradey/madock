package configs

import "strings"

// derivedKeys lists config values that are computed from another value rather
// than configured. A derived key stored in config.xml can only go stale, and a
// hand-set one can only be silently discarded — which is what happened with
// nodejs/major_version: twelve platform presets wrote it at setup, two
// generators recomputed it while rendering, and `config:set nodejs/major_version`
// looked accepted and changed nothing. The map points at the key that decides.
var derivedKeys = map[string]string{
	"nodejs/major_version": "nodejs/version",
}

// IsDerived reports whether an option is computed, and names the key that
// governs it.
func IsDerived(name string) (string, bool) {
	source, ok := derivedKeys[name]
	return source, ok
}

// applyDerived fills the computed keys from their sources. It runs on every
// assembled project config, so no caller has to remember to do it and no
// generator can render a stale value.
func applyDerived(conf map[string]string) {
	if conf == nil {
		return
	}

	// The NodeSource install script is addressed by major version only
	// (setup_20.x), while the image tag is the full one.
	if version := strings.TrimSpace(conf["nodejs/version"]); version != "" {
		conf["nodejs/major_version"] = strings.Split(version, ".")[0]
	} else {
		// No source, so any value present is left over from an older config.
		// Dropping it leaves the placeholder unsubstituted, which fails the
		// image build loudly; keeping it would install some Node nobody asked
		// for and look deliberate.
		delete(conf, "nodejs/major_version")
	}
}
