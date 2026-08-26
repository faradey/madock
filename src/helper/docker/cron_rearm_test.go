package docker

import "testing"

// Restarting the container the application runs in takes cron with it: no
// application image starts the daemon, so it exists only for as long as the
// container that madock started it in. The crontab survives, which is what made
// this quiet — the jobs are still installed and nothing runs them.
func TestCronIsRearmedWhenTheApplicationContainerRestarts(t *testing.T) {
	conf := map[string]string{
		"platform":     "custom",
		"language":     "nodejs",
		"cron/enabled": "true",
	}

	if !cronNeedsRearm(conf, []string{"nodejs"}) {
		t.Error("a Node project's cron was not rearmed after its own container restarted")
	}
	if !cronNeedsRearm(conf, []string{"nginx", "nodejs", "worker-queue"}) {
		t.Error("the main service was in the list and cron was still not rearmed")
	}
}

// The php fallback is the one every Magento and Shopware project takes, and it
// is reached by a project whose config names no language at all.
func TestCronIsRearmedForAPhpProject(t *testing.T) {
	conf := map[string]string{"platform": "magento2", "cron/enabled": "true"}

	if !cronNeedsRearm(conf, []string{"php"}) {
		t.Error("a Magento project's cron was not rearmed after the php container restarted")
	}
}

// Restarting something else must not touch cron. nginx has no cron daemon in it
// to lose, and `service:restart` exists precisely to be narrow — a restart that
// quietly does more than it was asked is the failure it was built against.
func TestCronIsLeftAloneWhenAnotherServiceRestarts(t *testing.T) {
	conf := map[string]string{
		"platform":     "custom",
		"language":     "nodejs",
		"cron/enabled": "true",
	}

	if cronNeedsRearm(conf, []string{"nginx"}) {
		t.Error("restarting nginx started cron; the restart is supposed to be precise")
	}
	if cronNeedsRearm(conf, nil) {
		t.Error("a restart of nothing started cron")
	}
}

// A project that never asked for cron does not get one. The rearm restores what
// the restart destroyed; it does not decide that a scheduler would be nice.
func TestCronIsNotStartedForAProjectThatDisabledIt(t *testing.T) {
	for _, value := range []string{"false", "", "0"} {
		conf := map[string]string{
			"platform":     "custom",
			"language":     "nodejs",
			"cron/enabled": value,
		}
		if cronNeedsRearm(conf, []string{"nodejs"}) {
			t.Errorf("cron/enabled=%q started cron after a restart", value)
		}
	}
}

// The flag is written by hand into a committed .madock/config.xml as often as
// it is written by `cron:enable`, so its spelling is not guaranteed to be the
// one madock writes.
func TestCronEnabledIsReadCaseInsensitively(t *testing.T) {
	conf := map[string]string{
		"platform":     "custom",
		"language":     "nodejs",
		"cron/enabled": "True",
	}

	if !cronNeedsRearm(conf, []string{"nodejs"}) {
		t.Error(`cron/enabled="True" was read as off`)
	}
}
