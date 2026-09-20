package install

import (
	"strings"
	"testing"
)

// The install chain is joined with &&, so one refusing command fails the whole
// run — after the store is already installed. `es:index` is that command when
// no search engine was chosen: Shopware answers "Elasticsearch indexing is
// disabled" and exits 1, and madock declared a finished install failed.
// Measured 2026-09-21, --search-engine=none, Shopware 6.7.14.1.
func TestShopwareInstallSkipsIndexingWithoutASearchEngine(t *testing.T) {
	conf := map[string]string{
		"search/elasticsearch/enabled": "false",
		"search/opensearch/enabled":    "false",
	}

	cmd := shopwareInstallCommand(conf, "shop.test", false)

	if strings.Contains(cmd, "es:index") {
		t.Errorf("es:index is run with no search engine, and refuses there:\n%s", cmd)
	}
	if strings.Contains(cmd, "SHOPWARE_ES_ENABLED=1") {
		t.Errorf("the .env still turns Elasticsearch on with no search engine:\n%s", cmd)
	}
	if !strings.Contains(cmd, "bin/console system:install") {
		t.Errorf("the install itself is missing from the chain:\n%s", cmd)
	}
}

// Both engines share the same env keys and the same indexing command, so one
// table covers them — and guards the other direction: skipping the index where
// an engine is configured would leave a store whose search returns nothing.
func TestShopwareInstallIndexesWithASearchEngine(t *testing.T) {
	cases := map[string]map[string]string{
		"elasticsearch": {"search/elasticsearch/enabled": "true", "search/opensearch/enabled": "false"},
		"opensearch":    {"search/elasticsearch/enabled": "false", "search/opensearch/enabled": "true"},
	}

	for name, conf := range cases {
		t.Run(name, func(t *testing.T) {
			cmd := shopwareInstallCommand(conf, "shop.test", false)

			if !strings.Contains(cmd, "&& bin/console es:index") {
				t.Errorf("a store with %s configured is not indexed:\n%s", name, cmd)
			}
			if !strings.Contains(cmd, "SHOPWARE_ES_ENABLED=1") {
				t.Errorf("the .env does not enable the engine that was chosen:\n%s", cmd)
			}
			// Indexing needs the shop to exist; order inside the chain is the
			// whole point of a chain.
			if strings.Index(cmd, "es:index") < strings.Index(cmd, "system:install") {
				t.Errorf("es:index is run before system:install:\n%s", cmd)
			}
		})
	}
}

func TestShopwareInstallAddsDemoDataOnlyWhenAsked(t *testing.T) {
	conf := map[string]string{"search/opensearch/enabled": "true"}

	with := shopwareInstallCommand(conf, "shop.test", true)
	without := shopwareInstallCommand(conf, "shop.test", false)

	if !strings.Contains(with, "framework:demodata") {
		t.Errorf("sample data was asked for and is not in the chain:\n%s", with)
	}
	if strings.Contains(without, "framework:demodata") {
		t.Errorf("sample data was not asked for and is in the chain:\n%s", without)
	}
	// Demo data must land before the index is built, or the index misses it.
	if strings.Index(with, "framework:demodata") > strings.Index(with, "es:index") {
		t.Errorf("demo data is generated after es:index, so it is not indexed:\n%s", with)
	}
}
