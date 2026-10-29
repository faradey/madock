package configs

import (
	"reflect"
	"strings"
	"testing"
)

// A project that commits search/engine believes it chose its engine. Every
// scope that declares it is named, and nothing else is.
func TestRetiredInNamesSearchEngineInEveryScope(t *testing.T) {
	stored := map[string]string{
		"activeScope":                              "default",
		"scopes/default/search/engine":             "opensearch",
		"scopes/default/search/opensearch/enabled": "true",
		"scopes/staging/search/engine":             "elasticsearch",
		// The snapshot engine is a live setting with the same leaf name.
		"scopes/default/snapshot/engine": "zfs",
	}

	want := []RetiredKey{
		{Key: "search/engine", Scope: "default", Instead: retiredKeys["search/engine"]},
		{Key: "search/engine", Scope: "staging", Instead: retiredKeys["search/engine"]},
	}
	if got := retiredIn(stored); !reflect.DeepEqual(got, want) {
		t.Fatalf("retired = %+v, want %+v", got, want)
	}

	delete(stored, "scopes/default/search/engine")
	delete(stored, "scopes/staging/search/engine")
	if got := retiredIn(stored); got != nil {
		t.Fatalf("retired = %+v, want none", got)
	}
}

// The default came back once, when two default files were made one, and
// `config:list` then reported elasticsearch on every project.
func TestARetiredKeyIsNotADefault(t *testing.T) {
	defaults := ParseXmlBytes(GetDefaultConfigXML())
	for key := range retiredKeys {
		for path := range defaults {
			if strings.HasSuffix(path, "/"+key) {
				t.Errorf("%s is retired and still a default (%s)", key, path)
			}
		}
		if _, retired := RetiredInstead(key); !retired {
			t.Errorf("RetiredInstead(%q) does not know it", key)
		}
	}
}
