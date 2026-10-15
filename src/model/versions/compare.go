package versions

import (
	"regexp"
	"strconv"
	"strings"
)

// Every provider here used to decide versions with `mageVer >= "2.4.9"`, which
// compares bytes, not versions. It held only while every segment stayed a single
// digit: "2.4.10" sorts below "2.4.9", so the first two-digit Magento release
// would have been handed the PHP, database and search engine of 2.4.0, and
// "2.4.7-p10" sorted below "2.4.7-p5".
//
// configs.CompareVersions and tmpl.CompareVersions do not fit either: both read
// "7-p5" through Atoi, get 0, and so put every Magento patch release below the
// release it patches. This package may not import helpers anyway.

var patchSuffix = regexp.MustCompile(`^p(\d+)$`)

type parsed struct {
	segments []int
	patch    int
}

// parse reads "2.4.7", "2.4.7-p5", "v6.7.13.1". A Magento "-pN" is a patch on
// top of the release and sorts after it. Any other suffix ("-beta1", "-rc")
// counts as the release itself, which is what the string comparison it replaces
// gave them and what a beta needs: the tool versions of the line it previews.
func parse(v string) (parsed, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if v == "" {
		return parsed{}, false
	}

	base, suffix, _ := strings.Cut(v, "-")

	var p parsed
	for _, part := range strings.Split(base, ".") {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return parsed{}, false
		}
		p.segments = append(p.segments, n)
	}

	if m := patchSuffix.FindStringSubmatch(suffix); m != nil {
		p.patch, _ = strconv.Atoi(m[1])
	}

	return p, true
}

// Compare returns 1 if a > b, -1 if a < b and 0 if they are equal. A missing
// segment counts as zero, so "6.5" and "6.5.0" are equal. Something that is not
// a version sorts below every version, and two of them are equal.
func Compare(a, b string) int {
	pa, okA := parse(a)
	pb, okB := parse(b)
	switch {
	case !okA && !okB:
		return 0
	case !okA:
		return -1
	case !okB:
		return 1
	}

	length := len(pa.segments)
	if len(pb.segments) > length {
		length = len(pb.segments)
	}
	for i := 0; i < length; i++ {
		var x, y int
		if i < len(pa.segments) {
			x = pa.segments[i]
		}
		if i < len(pb.segments) {
			y = pb.segments[i]
		}
		if x != y {
			if x > y {
				return 1
			}
			return -1
		}
	}

	if pa.patch != pb.patch {
		if pa.patch > pb.patch {
			return 1
		}
		return -1
	}

	return 0
}

// AtLeast reports whether v is min or later. An empty or unreadable v is never
// at least anything, as "" >= "2.0.0" was not.
func AtLeast(v, min string) bool {
	if _, ok := parse(v); !ok {
		return false
	}

	return Compare(v, min) >= 0
}
