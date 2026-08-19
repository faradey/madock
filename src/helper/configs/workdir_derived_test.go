package configs

import "testing"

// Where the application lives is derived, not stored, and this is the property
// that matters: the answer is right even when the file that used to hold it is
// not there.
//
// That window is not hypothetical. The pre-deploy rebuild stages the incoming
// `.madock` and, on every project whose config.xml is untracked, that tree has
// no config — so the project copy is gone while a rebuild reads it. Measured on
// 2026-08-19: a scheduler ran the wrong command for seventeen minutes on a live
// application because a stored copy showed through in that gap.
func TestWorkdirFollowsDeployWithNothingStored(t *testing.T) {
	conf := map[string]string{"deploy/enabled": "true"}

	applyDerived(conf)

	if got := conf["workdir"]; got != "/var/www/html/current" {
		t.Errorf("workdir = %q, want /var/www/html/current", got)
	}
}

// Without deployer the answer is the plain root, and a consumer that asks does
// not have to know which kind of project it is looking at.
func TestWorkdirWithoutDeployIsLeftAlone(t *testing.T) {
	conf := map[string]string{"workdir": "/var/www/html"}

	applyDerived(conf)

	if got := conf["workdir"]; got != "/var/www/html" {
		t.Errorf("workdir = %q, want /var/www/html", got)
	}
}

// Installations that enabled deploy before this was derived have the release
// path sitting in their config. Reading it must produce the same path, not
// current/current.
func TestWorkdirDerivationIsIdempotent(t *testing.T) {
	conf := map[string]string{"deploy/enabled": "true", "workdir": "/var/www/html/current"}

	applyDerived(conf)
	applyDerived(conf)

	if got := conf["workdir"]; got != "/var/www/html/current" {
		t.Errorf("workdir = %q, want /var/www/html/current", got)
	}
}

// A project whose application is not at the mount root keeps its own root and
// gains the release link, rather than losing the first to the second.
func TestWorkdirKeepsACustomRoot(t *testing.T) {
	conf := map[string]string{"deploy/enabled": "true", "workdir": "/var/www/html/storefront"}

	applyDerived(conf)

	if got := conf["workdir"]; got != "/var/www/html/storefront/current" {
		t.Errorf("workdir = %q, want /var/www/html/storefront/current", got)
	}
}

// php/workdir follows when it is set and is not invented when it is not: an
// unset one means the php container follows workdir, and writing one here would
// fix in place a thing that was deliberately left to follow.
func TestPhpWorkdirFollowsOnlyWhenSet(t *testing.T) {
	set := map[string]string{"deploy/enabled": "true", "php/workdir": "/var/www/html"}
	applyDerived(set)
	if got := set["php/workdir"]; got != "/var/www/html/current" {
		t.Errorf("php/workdir = %q, want /var/www/html/current", got)
	}

	unset := map[string]string{"deploy/enabled": "true"}
	applyDerived(unset)
	if _, exists := unset["php/workdir"]; exists {
		t.Errorf("php/workdir was invented: %q", unset["php/workdir"])
	}
}

// Trailing slashes and spaces come from hand-edited files, and neither should
// produce a different answer.
func TestWorkdirDerivationToleratesUntidyInput(t *testing.T) {
	for _, stored := range []string{"/var/www/html/", "  /var/www/html  ", "/var/www/html/current/"} {
		conf := map[string]string{"deploy/enabled": "true", "workdir": stored}
		applyDerived(conf)
		if got := conf["workdir"]; got != "/var/www/html/current" {
			t.Errorf("stored %q derived to %q", stored, got)
		}
	}
}

// Only "true" enables it. A half-configured deploy block must not move the
// application root.
func TestWorkdirIgnoresANonTrueDeployFlag(t *testing.T) {
	for _, flag := range []string{"", "false", "1", "yes"} {
		conf := map[string]string{"deploy/enabled": flag, "workdir": "/var/www/html"}
		applyDerived(conf)
		if got := conf["workdir"]; got != "/var/www/html" {
			t.Errorf("deploy/enabled=%q moved the root to %q", flag, got)
		}
	}
}
