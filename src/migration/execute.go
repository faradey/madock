package migration

import "github.com/faradey/madock/src/migration/versions"

func Execute(oldAppVersion string) {
	if oldAppVersion < "1.4.0" {
		versions.V140()
	} else if oldAppVersion < "1.8.0" {
		versions.V180()
	} else if oldAppVersion < "2.2.0" {
		versions.V220()
	}
}
