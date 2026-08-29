package docker

import (
	"os/exec"
	"strings"
	"sync"

	configs2 "github.com/faradey/madock/v4/src/helper/configs"
)

// The compose version, asked once and only when something depends on it.
//
// madock declares no minimum docker or compose version anywhere — not in the
// README, not in code — so every feature added to compose after 2021 is a
// feature we cannot assume. That was fine while nothing needed one, and stopped
// being fine when `docker compose pull` turned out to contact the registry even
// for an image already on the machine: the flag that fixes it, `--policy`,
// arrived in compose v2.22.0 on 2023-09-21.
//
// Passing an unknown flag is not a degradation, it is a hard failure — compose
// exits with "unknown flag" and `madock start` dies on somebody's machine over
// an optimisation that saves one request. So it is asked rather than assumed,
// and when the answer cannot be had the old behaviour stands.
const composePullPolicySince = "2.22.0"

var (
	composeVersionOnce sync.Once
	composeVersion     string
)

// ComposeVersion is the version of the compose plugin on this machine, or empty
// when it could not be determined.
//
// Empty is a real answer and is treated as "assume nothing", never as "old" or
// "new": docker may be absent, the plugin may be a fork, and the output format
// has changed before. A caller that cannot tell must keep doing what it did.
func ComposeVersion() string {
	composeVersionOnce.Do(func() {
		out, err := exec.Command("docker", "compose", "version", "--short").Output()
		if err != nil {
			return
		}
		// `--short` prints "2.29.7" on most builds and "v2.29.7" on some.
		composeVersion = strings.TrimPrefix(strings.TrimSpace(string(out)), "v")
	})

	return composeVersion
}

// composeSupportsPullPolicy reports whether `docker compose pull` here takes
// `--policy`.
func composeSupportsPullPolicy() bool {
	return versionHasPullPolicy(ComposeVersion())
}

// versionHasPullPolicy is the decision on its own, so it can be tested without
// a docker daemon — the part worth testing is the comparison, and the part that
// needs a machine is not.
func versionHasPullPolicy(version string) bool {
	if version == "" {
		return false
	}

	return configs2.CompareVersions(version, composePullPolicySince) >= 0
}
