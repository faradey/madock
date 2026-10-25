package setup

import (
	"strings"
	"testing"
)

// Without advisories the line is exactly what it always was. With them,
// create-project only writes the files, the ids go into the project's own
// composer.json, and the install after it reads them — the order is the
// point, since create-project resolves before any composer.json exists.
func TestAcceptedAdvisoriesReachTheProjectsComposerJSONBeforeTheInstall(t *testing.T) {
	plain := downloadScript("/var/www/html", "community", "2.4.8", false, nil)
	if strings.Contains(plain, "--no-install") || strings.Contains(plain, "audit.ignore") {
		t.Errorf("no advisories, and the line changed: %s", plain)
	}

	got := downloadScript("/var/www/html", "community", "2.4.7", false, []string{"PKSA-w9tt-7782-78jx"})
	create := strings.Index(got, "create-project --no-install")
	config := strings.Index(got, `audit.ignore --json '["PKSA-w9tt-7782-78jx"]'`)
	install := strings.LastIndex(got, "&& composer install")
	if create < 0 || config < create || install < config {
		t.Errorf("want create-project --no-install, then the ids, then the install:\n%s", got)
	}
	if !strings.Contains(got, "(cd ./download-magento123456789 && composer config") {
		t.Errorf("the ids are not written into the downloaded project's composer.json:\n%s", got)
	}
}
