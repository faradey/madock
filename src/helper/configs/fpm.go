package configs

import (
	"fmt"
	"strconv"
	"strings"
)

// fpmPool is the set of pool keys that have to agree with each other.
var fpmPool = []string{
	"php/fpm/max_children",
	"php/fpm/start_servers",
	"php/fpm/min_spare_servers",
	"php/fpm/max_spare_servers",
}

// ValidateFpmPool reports a pool php-fpm would refuse to start with.
//
// The four values are independent settings and not independent numbers: php-fpm
// checks them against each other at start-up and exits if they disagree. Lower
// `php/fpm/max_children` to 2 and leave the spare bounds where they are, and it
// answers
//
//	pm.max_spare_servers(3) must not be greater than pm.max_children(2)
//
// which is a good message in a bad place — it arrives from inside the container,
// after a full image rebuild, on a project that was working ten minutes ago. The
// same disagreement is visible the moment the file is read, so it is said here
// instead, before anything is built.
//
// Deliberately not clamping the numbers into something valid. A pool quietly
// adjusted to what madock thinks the author meant is a configuration that says
// one thing and runs another, and the next person to read it has no way to tell.
func ValidateFpmPool(conf map[string]string) error {
	values := map[string]int{}

	for _, key := range fpmPool {
		raw, ok := conf[key]
		if !ok {
			// Absent — the embedded defaults answer for it, and those agree.
			continue
		}

		// Present and empty is a different thing, and it used to be waved
		// through here on the same reasoning as absent. It is not the same: the
		// defaults only fill keys that are *missing* (ConfigMapping skips any
		// key already in the map, whatever its value), so an empty one survives
		// all the way into the template and renders `pm.max_children = `. Then
		// php-fpm refuses to start, from inside the container, after a full
		// image build — which is the answer this function exists to move
		// earlier.
		if strings.TrimSpace(raw) == "" {
			return fmt.Errorf("%s is set to nothing. Remove the key to take the default, or give it a number — "+
				"an empty one reaches php-fpm as `pm.%s = ` and it refuses to start",
				key, strings.TrimPrefix(key, "php/fpm/"))
		}

		value, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("%s is %q, which is not a number of processes", key, raw)
		}
		if value < 1 {
			return fmt.Errorf("%s is %d; php-fpm needs at least one", key, value)
		}

		values[key] = value
	}

	if len(values) != len(fpmPool) {
		// Some keys are absent entirely and the defaults answer for those. What
		// this installation will actually run is a mixture of the two, and
		// comparing a configured number against a default it may not be using
		// would refuse a pool that works.
		return nil
	}

	maxChildren := values["php/fpm/max_children"]
	start := values["php/fpm/start_servers"]
	minSpare := values["php/fpm/min_spare_servers"]
	maxSpare := values["php/fpm/max_spare_servers"]

	if maxSpare > maxChildren {
		return fmt.Errorf("php/fpm/max_spare_servers is %d and php/fpm/max_children is %d — php-fpm refuses to start "+
			"with more spare workers than workers", maxSpare, maxChildren)
	}
	if minSpare > maxSpare {
		return fmt.Errorf("php/fpm/min_spare_servers is %d and php/fpm/max_spare_servers is %d — the floor is above "+
			"the ceiling", minSpare, maxSpare)
	}
	if start < minSpare || start > maxSpare {
		return fmt.Errorf("php/fpm/start_servers is %d, which is outside php/fpm/min_spare_servers (%d) and "+
			"php/fpm/max_spare_servers (%d) — php-fpm refuses to start", start, minSpare, maxSpare)
	}

	return nil
}
