package tools

import (
	"testing"

	"github.com/faradey/madock/v4/src/model/versions"
)

// withReconfigure sets the package-level mode for one test and restores it, so
// the tests cannot leak state into each other.
func withReconfigure(t *testing.T, v bool) {
	t.Helper()
	saved := reconfigureMode
	reconfigureMode = v
	t.Cleanup(func() { reconfigureMode = saved })
}

// On a first-time setup the caller has no project config to offer, so it passes
// the general config — which starts from the embedded defaults. Reading versions
// out of that would replace the platform's version matrix with values nobody
// chose. A Shopware 6.7 project ended up pinned to the default OpenSearch
// version instead of the 2.8.0 its matrix names, and the first start failed
// pulling a tag that does not exist.
func TestPopulateFromConfigIgnoredOutsideReconfigure(t *testing.T) {
	withReconfigure(t, false)

	tv := versions.ToolsVersions{Php: "8.3", OpenSearch: "2.8.0", Elastic: "8.11.14"}
	PopulateFromConfig(&tv, map[string]string{
		"php/version":                  "8.1",
		"search/opensearch/version":    "2.5",
		"search/elasticsearch/version": "8.4.3",
	})

	if tv.OpenSearch != "2.8.0" {
		t.Errorf("OpenSearch = %q, want the matrix value 2.8.0", tv.OpenSearch)
	}
	if tv.Elastic != "8.11.14" {
		t.Errorf("Elastic = %q, want the matrix value 8.11.14", tv.Elastic)
	}
	if tv.Php != "8.3" {
		t.Errorf("Php = %q, want the matrix value 8.3", tv.Php)
	}
}

// Reconfiguring an existing project is the case the function exists for: what
// the project already runs wins over what the matrix would suggest.
func TestPopulateFromConfigAppliedInReconfigure(t *testing.T) {
	withReconfigure(t, true)

	tv := versions.ToolsVersions{Php: "8.3", OpenSearch: "2.8.0"}
	PopulateFromConfig(&tv, map[string]string{
		"php/version":               "8.1",
		"search/opensearch/version": "2.19.1",
	})

	if tv.OpenSearch != "2.19.1" {
		t.Errorf("OpenSearch = %q, want the configured 2.19.1", tv.OpenSearch)
	}
	if tv.Php != "8.1" {
		t.Errorf("Php = %q, want the configured 8.1", tv.Php)
	}
}

// An empty value in the config means "not set", not "set to nothing" — it must
// not blank out a matrix value.
func TestPopulateFromConfigKeepsMatrixForEmptyValues(t *testing.T) {
	withReconfigure(t, true)

	tv := versions.ToolsVersions{OpenSearch: "2.8.0"}
	PopulateFromConfig(&tv, map[string]string{"search/opensearch/version": ""})

	if tv.OpenSearch != "2.8.0" {
		t.Errorf("OpenSearch = %q, want 2.8.0 kept", tv.OpenSearch)
	}
}
