package docker

import "strings"

// The crontab of www-data is not ours alone. Magento writes a
// `#~ MAGENTO START ... #~ MAGENTO END` block into it through `cron:install`,
// Shopware appends a `scheduled-task:run` line, and a person may have added
// something by hand inside the container. Config jobs used to be installed by
// wiping the whole crontab and writing them over it, which worked only because
// the platform branch that runs immediately afterwards happened to put its own
// block back — and `cron:install` is skipped whenever Magento's DI is still
// warming up, in which case nothing did.
//
// So the config's jobs get a block of their own, and every read-modify-write
// touches nothing outside it.
const (
	cronBlockStart = "#~ MADOCK START"
	cronBlockEnd   = "#~ MADOCK END"
)

// mergeCrontab returns the crontab that should replace `existing` once the
// config's jobs are installed: everything that is not ours, followed by our
// block.
//
// Lines identical to a job being installed are dropped from the foreign part
// too. That is not tidiness — before this block existed, config jobs were
// written into the crontab unmarked, so on the first run after an upgrade they
// are sitting there with nothing to identify them, and appending the block
// without this would double every one of them.
func mergeCrontab(existing string, jobs []string) string {
	kept := stripMadockBlock(existing)

	installing := make(map[string]bool, len(jobs))
	for _, job := range jobs {
		installing[strings.TrimSpace(job)] = true
	}

	var foreign []string
	for _, line := range kept {
		if installing[strings.TrimSpace(line)] {
			continue
		}
		foreign = append(foreign, line)
	}

	if len(jobs) == 0 {
		return joinCrontab(foreign)
	}

	block := append([]string{cronBlockStart}, jobs...)
	block = append(block, cronBlockEnd)
	return joinCrontab(append(foreign, block...))
}

// removeMadockBlock returns the crontab with our block taken out and everything
// else left where it was.
func removeMadockBlock(existing string) string {
	return joinCrontab(stripMadockBlock(existing))
}

// stripMadockBlock drops the lines between our markers, inclusive, and returns
// what is left.
//
// An unterminated start marker takes the rest of the file with it: the block is
// ours to the end, and leaving its contents behind would reinstall them as
// foreign lines on the next merge.
func stripMadockBlock(existing string) []string {
	var kept []string
	inBlock := false
	for _, line := range strings.Split(existing, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == cronBlockStart {
			inBlock = true
			continue
		}
		if trimmed == cronBlockEnd {
			inBlock = false
			continue
		}
		if inBlock {
			continue
		}
		kept = append(kept, line)
	}

	// Trailing blank lines accumulate over rewrites otherwise.
	for len(kept) > 0 && strings.TrimSpace(kept[len(kept)-1]) == "" {
		kept = kept[:len(kept)-1]
	}
	return kept
}

// Magento marks its own block `#~ MAGENTO START <sha256(BP)> … #~ MAGENTO END
// <sha256(BP)>`, where BP is the installation's base path. `cron:remove` finds
// the block by recomputing that hash from where it is run — so on a deployer
// layout, where BP is `…/releases/<n>` and every release is a new directory, the
// command run from the new release cannot see the previous release's block and
// reports success without touching it.
//
// Measured on extmag.com on 2026-08-27: after one deploy the crontab carried two
// MAGENTO blocks, `releases/159` and `releases/160`, so `cron:run` started twice
// a minute out of two trees. The second half is the one that bites later:
// `deploy:cleanup` removes `releases/159`, and the entry stays behind pointing
// into nothing — a php that fails every minute into a shared log.
//
// Magento cannot fix this by construction, so the cleanup is madock's.
const (
	magentoBlockStart = "#~ MAGENTO START"
	magentoBlockEnd   = "#~ MAGENTO END"
)

// stripStaleMagentoBlocks removes every Magento block that does not belong to
// the installation now running, and leaves everything else alone.
//
// `keepPath` is the base path of the current installation, resolved through the
// `current` symlink — the block to keep is the one whose command names it.
// A block is dropped only when it names some *other* absolute path under the
// same parent: a crontab written by a Magento that madock does not manage, or a
// line whose shape is not understood, is not ours to delete.
func stripStaleMagentoBlocks(existing, keepPath string) (string, []string) {
	if strings.TrimSpace(keepPath) == "" {
		return existing, nil
	}

	var kept, dropped []string
	var block []string
	inBlock := false
	blockNamesKeepPath := false

	for _, line := range strings.Split(existing, "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, magentoBlockStart):
			inBlock = true
			blockNamesKeepPath = false
			block = []string{line}
			continue
		case strings.HasPrefix(trimmed, magentoBlockEnd) && inBlock:
			block = append(block, line)
			if blockNamesKeepPath {
				kept = append(kept, block...)
			} else {
				dropped = append(dropped, block...)
			}
			inBlock = false
			block = nil
			continue
		}

		if inBlock {
			block = append(block, line)
			if strings.Contains(line, keepPath) {
				blockNamesKeepPath = true
			}
			continue
		}

		kept = append(kept, line)
	}

	// A start marker with no end takes the rest with it, and is kept rather than
	// dropped: an unterminated block is damage of some other kind, and removing
	// what is left of it would hide it.
	if inBlock {
		kept = append(kept, block...)
	}

	for len(kept) > 0 && strings.TrimSpace(kept[len(kept)-1]) == "" {
		kept = kept[:len(kept)-1]
	}

	if len(dropped) == 0 {
		return existing, nil
	}
	return joinCrontab(kept), dropped
}

func joinCrontab(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// writeCrontabScript builds the shell that replaces www-data's crontab with
// `content`.
//
// A quoted heredoc rather than `echo '...'`: the previous spelling put the job
// text inside single quotes, so a job containing one ended the quoting and the
// rest of the command was whatever the shell made of it. These are arbitrary
// strings out of somebody's config, and there is no reason a command line
// cannot contain a quote.
func writeCrontabScript(content string) string {
	if strings.TrimSpace(content) == "" {
		// An empty crontab is removed rather than written as a blank file, so
		// `crontab -l` answers "no crontab for www-data" instead of nothing.
		return "crontab -u www-data -r 2>/dev/null || true"
	}

	delimiter := "MADOCK_CRONTAB_EOF"
	for lineContains(content, delimiter) {
		delimiter += "_"
	}

	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return "cat <<'" + delimiter + "' | crontab -u www-data -\n" + content + delimiter + "\n"
}

// lineContains reports whether any line of s is exactly delimiter, which is the
// only way a heredoc can be ended early.
func lineContains(s, delimiter string) bool {
	for _, line := range strings.Split(s, "\n") {
		if line == delimiter {
			return true
		}
	}
	return false
}
