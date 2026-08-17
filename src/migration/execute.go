package migration

import (
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/migration/versions"
)

// olderThan compares two versions as versions rather than as strings.
//
// They used to be compared with `<`, which is right for every version this
// project has had and wrong for the next one: "3.9.10" < "3.9.8" is **true**
// as a string, because '1' sorts before '8'. The first two-digit patch release
// therefore re-ran a migration on every command, forever, and would have gone
// on doing so quietly — the migrations are written to be harmless when there is
// nothing to do, which is exactly what would have kept anybody from noticing.
//
// Found while tagging 3.9.10, before it reached anything.
func olderThan(version, than string) bool {
	return configs.CompareVersions(version, than) < 0
}

func Execute(oldAppVersion string) {
	if olderThan(oldAppVersion, "1.4.0") {
		versions.V140()
	}
	if olderThan(oldAppVersion, "1.8.0") {
		versions.V180()
	}
	if olderThan(oldAppVersion, "2.2.0") {
		versions.V220()
	}
	if olderThan(oldAppVersion, "2.3.0") {
		versions.V230()
	}
	if olderThan(oldAppVersion, "2.4.0") {
		versions.V240()
	}
	if olderThan(oldAppVersion, "3.1.0") {
		versions.V310()
	}
	if olderThan(oldAppVersion, "3.2.0") {
		versions.V320()
	}
	if olderThan(oldAppVersion, "3.3.0") {
		versions.V330()
	}
	if olderThan(oldAppVersion, "3.4.0") {
		versions.V340()
	}
	if olderThan(oldAppVersion, "3.5.9") {
		versions.V359()
	}
	if olderThan(oldAppVersion, "3.6.7") {
		versions.V366()
	}
	if olderThan(oldAppVersion, "3.7.2") {
		versions.V372()
	}
	if olderThan(oldAppVersion, "3.7.5") {
		versions.V375()
	}
	if olderThan(oldAppVersion, "3.8.5") {
		versions.V385()
	}
	if olderThan(oldAppVersion, "3.9.8") {
		versions.V398()
	}
}
