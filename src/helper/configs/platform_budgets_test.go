package configs

import (
	"testing"

	"github.com/faradey/madock/v4/src/model/versions"
)

// The order is madock's defaults, the platform's, projects/config.xml, the
// project. A platform value replaces a default and never a value anybody set.
func TestAPlatformBudgetReplacesTheDefaultAndNothingElse(t *testing.T) {
	saved := platformBudgets
	platformBudgets = func(platform string) map[string]string {
		if platform != "magento2" {
			return nil
		}
		return map[string]string{"search/opensearch/heap": "2g", "db/memory": "1G"}
	}
	defer func() { platformBudgets = saved }()

	config := map[string]string{
		"platform":               "magento2",
		"search/opensearch/heap": "1g",   // madock's default
		"db/memory":              "512M", // set in the project
	}
	applyPlatformBudgets(config, map[string]bool{"db/memory": true})

	if got := config["search/opensearch/heap"]; got != "2g" {
		t.Errorf("the platform's heap should replace madock's default, got %q", got)
	}
	if got := config["db/memory"]; got != "512M" {
		t.Errorf("a value set in the project must win over the platform's, got %q", got)
	}

	other := map[string]string{"platform": "shopware", "search/opensearch/heap": "1g"}
	applyPlatformBudgets(other, map[string]bool{})
	if got := other["search/opensearch/heap"]; got != "1g" {
		t.Errorf("another platform must not get magento2's value, got %q", got)
	}
}

func embeddedDefaults(t *testing.T) map[string]string {
	t.Helper()
	if len(defaultConfigXML) == 0 {
		t.Fatal("no embedded defaults")
	}
	return getConfigByScope(ParseXmlBytes(defaultConfigXML), "default")
}

// A budget a platform may set has to be a real setting, or the value would
// apply to nothing.
func TestEveryBudgetKeyIsARealSetting(t *testing.T) {
	defaults := embeddedDefaults(t)
	for key := range versions.BudgetKeys {
		if _, ok := defaults[key]; !ok {
			t.Errorf("%q is not in config_defaults.xml", key)
		}
	}
}

// A platform value equal to madock's default is a second copy of it, and a
// copy stops following the default the day the default changes.
func TestNoPlatformRepeatsADefault(t *testing.T) {
	defaults := embeddedDefaults(t)
	for _, platform := range versions.MatrixPlatforms() {
		for key, value := range versions.PlatformBudgets(platform) {
			if defaults[key] == value {
				t.Errorf("matrix/%s.xml sets %s to %q, which is madock's default already — remove the line", platform, key, value)
			}
		}
	}
}
