package migration

import "testing"

// The gate that decides whether a migration runs used to be a string
// comparison, and the first two-digit patch release breaks it: "3.9.10" sorts
// before "3.9.8" because '1' sorts before '8'.
//
// That would have re-run a migration on every command, forever, and quietly:
// migrations here are written to be harmless when there is nothing to do, which
// is precisely what would have stopped anybody noticing. Found while tagging
// 3.9.10, before it reached anything.
func TestOlderThan(t *testing.T) {
	for _, c := range []struct {
		version, than string
		want          bool
	}{
		// The case that broke.
		{"3.9.10", "3.9.8", false},
		{"3.9.8", "3.9.10", true},
		{"3.10.0", "3.9.8", false},
		{"3.9.8", "3.10.0", true},

		// And the ordinary ones, which the string compare also got right —
		// they are here so a future rewrite cannot trade one for the other.
		{"3.8.5", "3.9.8", true},
		{"3.9.8", "3.9.8", false},
		{"3.9.9", "3.9.8", false},
		{"1.4.0", "3.9.8", true},
		{"2.4.0", "3.1.0", true},
	} {
		if got := olderThan(c.version, c.than); got != c.want {
			t.Errorf("olderThan(%q, %q) = %v, want %v", c.version, c.than, got, c.want)
		}
	}
}

// A run of migrations on an installation that is already current must do
// nothing at all. This is the property the string compare silently lost.
func TestNothingRunsOnACurrentInstallation(t *testing.T) {
	for _, version := range []string{"3.9.10", "3.10.0", "4.0.0"} {
		for _, gate := range []string{"1.4.0", "3.3.0", "3.8.5", "3.9.8"} {
			if olderThan(version, gate) {
				t.Errorf("an installation on %s would re-run the %s migration", version, gate)
			}
		}
	}
}

// The same comparison decides whether migrations run at all, and there it fails
// the other way: "3.9.8" < "3.9.10" is false as a string, so an installation on
// 3.9.8 upgrading to 3.9.10 would have run nothing — and the rename of
// php/nodejs/enabled would never have happened for anybody who had it.
//
// Silent in both directions. Nothing errors; the key is simply gone.
func TestAnUpgradeToATwoDigitPatchStillMigrates(t *testing.T) {
	for _, c := range []struct{ from, to string }{
		{"3.9.8", "3.9.10"},
		{"3.9.9", "3.9.11"},
		{"3.9.11", "3.10.0"},
	} {
		if !olderThan(c.from, c.to) {
			t.Errorf("upgrading %s -> %s would run no migrations", c.from, c.to)
		}
	}

	// And a downgrade, or the same version, must still run nothing.
	if olderThan("3.9.10", "3.9.10") {
		t.Error("the same version was treated as an upgrade")
	}
	if olderThan("3.10.0", "3.9.11") {
		t.Error("a downgrade was treated as an upgrade")
	}
}
