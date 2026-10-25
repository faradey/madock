// Package composerpolicy is how far madock lets Composer's advisory blocking
// be relaxed for one project.
//
// Composer 2.9 blocks packages with a security advisory on create-project,
// install and update. Since 2026-09-29 that blocks every Magento below 2.4.8:
// they need league/flysystem 1.x or 2.x, PKSA-w9tt-7782-78jx covers all of
// them, and no 1.x or 2.x release is fixed. Somebody keeping a 2.4.7 shop on
// their own machine still has every right to work on it, so setup can be told
// which advisories this one project accepts — named one by one, never all of
// them — and they are written into the project's own composer.json, where the
// decision stays visible and covers its later updates. madock Pro, which runs
// servers, refuses it through Refusal.
package composerpolicy

import (
	"fmt"
	"regexp"
	"strings"
)

// Refusal, when set, says why this edition does not let advisories be
// accepted; "" lets it. madock Pro sets it in an init.
var Refusal func() string

// advisoryID is what an advisory id looks like: packagist's PKSA-xxxx-xxxx-xxxx,
// a CVE, a GHSA. It is pasted into a shell command, so nothing else gets through.
var advisoryID = regexp.MustCompile(`^(PKSA-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}|CVE-\d{4}-\d{4,}|GHSA(-[a-z0-9]{4}){3})$`)

// Parse reads --allow-advisory: ids separated by commas or spaces. A value
// that is not an id is refused rather than dropped.
func Parse(raw string) ([]string, error) {
	var ids []string
	for _, f := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' }) {
		if !advisoryID.MatchString(f) {
			return nil, fmt.Errorf("%q is not an advisory id (PKSA-…, CVE-…, GHSA-…); accepting advisories goes by id, never all of them", f)
		}
		ids = append(ids, f)
	}
	return ids, nil
}

// ConfigCommand is the composer command that writes ids into the composer.json
// in the current directory, or "" for none.
func ConfigCommand(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return "composer config audit.ignore --json '[\"" + strings.Join(ids, `","`) + "\"]'"
}
