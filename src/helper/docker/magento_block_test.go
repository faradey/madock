package docker

import (
	"strings"
	"testing"
)

// The crontab measured on extmag.com on 2026-08-27, after one deploy: two
// Magento blocks, one per release, both running `cron:run` every minute out of
// their own tree. Magento cannot remove the older one — `cron:remove` finds a
// block by recomputing sha256 of the base path it is run from, and that path is
// the new release.
const twoReleaseCrontab = `#~ MAGENTO START 3563c154
* * * * * /usr/bin/php8.5 /var/www/html/releases/159/bin/magento cron:run >> /var/www/html/releases/159/var/log/magento.cron.log 2>&1
#~ MAGENTO END 3563c154
#~ MAGENTO START 1286826a
* * * * * /usr/bin/php8.5 /var/www/html/releases/160/bin/magento cron:run >> /var/www/html/releases/160/var/log/magento.cron.log 2>&1
#~ MAGENTO END 1286826a
`

func TestTheBlockOfAnEarlierReleaseIsRemoved(t *testing.T) {
	cleaned, dropped := stripStaleMagentoBlocks(twoReleaseCrontab, "/var/www/html/releases/160")

	if strings.Contains(cleaned, "releases/159") {
		t.Errorf("the previous release's job is still installed:\n%s", cleaned)
	}
	if !strings.Contains(cleaned, "releases/160") {
		t.Errorf("the current release's job was removed:\n%s", cleaned)
	}
	if len(dropped) == 0 {
		t.Error("nothing was reported as removed, so the operator is not told the schedule changed")
	}
	if strings.Count(cleaned, "#~ MAGENTO START") != 1 {
		t.Errorf("expected exactly one Magento block to survive:\n%s", cleaned)
	}
}

// Everything that is not Magento's stays where it is. The crontab is shared —
// madock's own block, a Shopware line, something added by hand in the container
// — and a cleanup that tidies more than it was asked to is worse than the
// duplicate it removes.
func TestForeignLinesSurviveTheCleanup(t *testing.T) {
	crontab := "#~ MADOCK START\n* * * * * /bin/true\n#~ MADOCK END\n" +
		twoReleaseCrontab +
		"*/5 * * * * /var/www/html/current/bin/console scheduled-task:run\n"

	cleaned, _ := stripStaleMagentoBlocks(crontab, "/var/www/html/releases/160")

	for _, line := range []string{"#~ MADOCK START", "/bin/true", "scheduled-task:run"} {
		if !strings.Contains(cleaned, line) {
			t.Errorf("the cleanup removed %q, which is not Magento's:\n%s", line, cleaned)
		}
	}
}

// A single block naming the current release is the ordinary state, and it must
// come back unchanged — byte for byte, so nothing rewrites the crontab on every
// start for no reason.
func TestASingleCurrentBlockIsLeftAlone(t *testing.T) {
	only := `#~ MAGENTO START 1286826a
* * * * * /usr/bin/php8.5 /var/www/html/releases/160/bin/magento cron:run
#~ MAGENTO END 1286826a
`

	cleaned, dropped := stripStaleMagentoBlocks(only, "/var/www/html/releases/160")

	if len(dropped) != 0 {
		t.Errorf("a healthy crontab was edited: %v", dropped)
	}
	if cleaned != only {
		t.Errorf("the crontab was rewritten when nothing had to change:\n%s", cleaned)
	}
}

// An unknown base path removes nothing. Guessing here would delete the only
// working schedule on a project whose layout this does not understand.
func TestAnUnknownBasePathRemovesNothing(t *testing.T) {
	cleaned, dropped := stripStaleMagentoBlocks(twoReleaseCrontab, "")

	if len(dropped) != 0 || cleaned != twoReleaseCrontab {
		t.Error("blocks were removed without knowing which installation is current")
	}
}

// Two blocks naming the same installation are a duplicate, not a pair.
//
// `cron:install` clears the previous block through the same function
// `cron:remove` uses, and that one cannot remove the last block in the file — so
// installing over an install appends instead of replacing, and both copies run
// `cron:run` every minute.
func TestADuplicateOfTheCurrentBlockIsRemoved(t *testing.T) {
	duplicated := `#~ MAGENTO START 1286826a
* * * * * /usr/bin/php8.5 /var/www/html/releases/160/bin/magento cron:run
#~ MAGENTO END 1286826a
#~ MAGENTO START 1286826a
* * * * * /usr/bin/php8.5 /var/www/html/releases/160/bin/magento cron:run
#~ MAGENTO END 1286826a
`

	cleaned, dropped := stripStaleMagentoBlocks(duplicated, "/var/www/html/releases/160")

	if strings.Count(cleaned, "cron:run") != 1 {
		t.Errorf("the duplicate is still installed and runs twice a minute:\n%s", cleaned)
	}
	if len(dropped) == 0 {
		t.Error("the duplicate was removed without saying so")
	}
}

// Disabling cron has to remove every Magento block, including the last one —
// which `bin/magento cron:remove` reports as removed and leaves in place.
func TestDisablingRemovesEveryMagentoBlock(t *testing.T) {
	crontab := "#~ MADOCK START\n* * * * * /bin/true\n#~ MADOCK END\n" + twoReleaseCrontab

	cleaned, dropped := removeAllMagentoBlocks(crontab)

	if strings.Contains(cleaned, "MAGENTO") {
		t.Errorf("a Magento block survived being disabled:\n%s", cleaned)
	}
	if len(dropped) == 0 {
		t.Error("nothing was reported as removed")
	}
	if !strings.Contains(cleaned, "/bin/true") {
		t.Errorf("disabling Magento's cron removed somebody else's job:\n%s", cleaned)
	}
}

// An unterminated block is damage of another kind. Removing what is left of it
// would hide that, so it is kept.
func TestAnUnterminatedBlockIsKept(t *testing.T) {
	broken := "#~ MAGENTO START abc\n* * * * * /var/www/html/releases/159/bin/magento cron:run\n"

	cleaned, dropped := stripStaleMagentoBlocks(broken, "/var/www/html/releases/160")

	if len(dropped) != 0 {
		t.Errorf("an unterminated block was silently removed: %v", dropped)
	}
	if !strings.Contains(cleaned, "releases/159") {
		t.Errorf("the contents of an unterminated block were dropped:\n%s", cleaned)
	}
}
