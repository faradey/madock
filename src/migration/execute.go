package migration

import "github.com/faradey/madock/src/migration/versions"

func Execute(oldAppVersion string) {
	if oldAppVersion < "1.4.0" {
		versions.V140()
	}
	if oldAppVersion < "1.8.0" {
		versions.V180()
	}
	if oldAppVersion < "2.2.0" {
		versions.V220()
	}
	if oldAppVersion < "2.3.0" {
		versions.V230()
	}
	if oldAppVersion < "2.4.0" {
		versions.V240()
	}
}
