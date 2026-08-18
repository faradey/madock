package docker

import (
	"strings"
	"testing"
)

const magentoBlock = `#~ MAGENTO START db8b1a
* * * * * /usr/bin/php /var/www/html/bin/magento cron:run 2>&1 | grep -v "Ran jobs by schedule"
#~ MAGENTO END db8b1a`

// The crontab is shared. Installing the config's jobs used to wipe it and write
// them over the top, which left Magento's block standing only because
// `cron:install` runs a moment later and puts it back — and that call is skipped
// whenever Magento's DI is still warming up, in which case nothing does.
func TestMergeCrontabKeepsForeignBlocks(t *testing.T) {
	existing := magentoBlock + "\n@reboot /opt/added-by-hand.sh\n"

	out := mergeCrontab(existing, []string{"* * * * * /a.sh"})

	for _, want := range []string{
		"#~ MAGENTO START db8b1a",
		"bin/magento cron:run",
		"#~ MAGENTO END db8b1a",
		"@reboot /opt/added-by-hand.sh",
		cronBlockStart,
		"* * * * * /a.sh",
		cronBlockEnd,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// Reinstalling replaces our block rather than stacking another one on top.
func TestMergeCrontabReplacesOwnBlock(t *testing.T) {
	first := mergeCrontab(magentoBlock, []string{"* * * * * /a.sh", "* * * * * /b.sh"})
	second := mergeCrontab(first, []string{"* * * * * /c.sh"})

	if strings.Count(second, cronBlockStart) != 1 {
		t.Errorf("expected exactly one block:\n%s", second)
	}
	if strings.Contains(second, "/a.sh") || strings.Contains(second, "/b.sh") {
		t.Errorf("previous jobs survived:\n%s", second)
	}
	if !strings.Contains(second, "/c.sh") || !strings.Contains(second, "MAGENTO START") {
		t.Errorf("lost the new job or the foreign block:\n%s", second)
	}
}

// Before the block existed, config jobs were written unmarked. On the first run
// after an upgrade they are indistinguishable from anything else in the file,
// and appending the block without noticing would double every one of them.
func TestMergeCrontabAdoptsUnmarkedJobsFromOlderVersions(t *testing.T) {
	existing := "* * * * * /a.sh\n* * * * * /b.sh\n" + magentoBlock

	out := mergeCrontab(existing, []string{"* * * * * /a.sh", "* * * * * /b.sh"})

	if got := strings.Count(out, "/a.sh"); got != 1 {
		t.Errorf("/a.sh appears %d times, want 1:\n%s", got, out)
	}
	if !strings.Contains(out, "MAGENTO START") {
		t.Errorf("lost the foreign block:\n%s", out)
	}
}

// An empty job list removes our block and nothing else.
func TestMergeCrontabWithNoJobsLeavesForeignLines(t *testing.T) {
	existing := mergeCrontab(magentoBlock, []string{"* * * * * /a.sh"})

	out := mergeCrontab(existing, nil)

	if strings.Contains(out, cronBlockStart) || strings.Contains(out, "/a.sh") {
		t.Errorf("our block survived:\n%s", out)
	}
	if !strings.Contains(out, "MAGENTO START") {
		t.Errorf("removed somebody else's block:\n%s", out)
	}
}

// A start marker with no end is ours to the end of the file. Leaving the
// contents behind would reinstate them as foreign lines on the next merge.
func TestStripMadockBlockHandlesUnterminatedBlock(t *testing.T) {
	existing := "@reboot /keep.sh\n" + cronBlockStart + "\n* * * * * /a.sh\n"

	out := removeMadockBlock(existing)

	if strings.Contains(out, "/a.sh") {
		t.Errorf("unterminated block survived:\n%s", out)
	}
	if !strings.Contains(out, "/keep.sh") {
		t.Errorf("dropped a line before the marker:\n%s", out)
	}
}

// Jobs are arbitrary strings out of somebody's config. `echo '...'` put them
// inside single quotes, so one quote in a command ended the quoting and the
// shell made of the rest whatever it liked.
func TestWriteCrontabScriptSurvivesQuotes(t *testing.T) {
	job := `* * * * * php -r 'echo "hi";' >> /var/log/x.log 2>&1`

	script := writeCrontabScript(mergeCrontab("", []string{job}))

	if !strings.Contains(script, "<<'MADOCK_CRONTAB_EOF'") {
		t.Errorf("expected a quoted heredoc:\n%s", script)
	}
	if !strings.Contains(script, job) {
		t.Errorf("job was mangled:\n%s", script)
	}
}

// A job line equal to the delimiter would end the heredoc early and feed the
// rest of the crontab to the shell.
func TestWriteCrontabScriptPicksAFreeDelimiter(t *testing.T) {
	script := writeCrontabScript("MADOCK_CRONTAB_EOF\n* * * * * /a.sh\n")

	if !strings.Contains(script, "<<'MADOCK_CRONTAB_EOF_'") {
		t.Errorf("delimiter collides with the content:\n%s", script)
	}
}

// Writing an empty file would leave `crontab -l` answering with nothing, which
// reads as an empty schedule rather than none at all.
func TestWriteCrontabScriptRemovesWhenEmpty(t *testing.T) {
	script := writeCrontabScript("")

	if !strings.Contains(script, "crontab -u www-data -r") {
		t.Errorf("expected a removal:\n%s", script)
	}
}
