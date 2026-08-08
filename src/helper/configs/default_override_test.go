package configs

import "testing"

// The layering is the whole point of this seam, so it is what gets tested: an
// edition may change a default, and the person editing config.xml still wins.
// Get that order wrong and turning a setting back on needs a different binary.
func TestDefaultOverrideSitsBetweenEmbeddedAndUserConfig(t *testing.T) {
	t.Cleanup(func() { defaultOverrides = map[string]string{} })

	embedded := ParseXmlBytes(defaultConfigXML)
	embedded = getConfigByScope(embedded, "default")

	if embedded["proxy/mailpit/enabled"] != "true" {
		t.Fatalf("this test assumes mailpit ships enabled; it is %q", embedded["proxy/mailpit/enabled"])
	}

	SetDefaultOverride("proxy/mailpit/enabled", "false")

	overridden := ParseXmlBytes(defaultConfigXML)
	overridden = getConfigByScope(overridden, "default")
	for key, value := range defaultOverrides {
		overridden[key] = value
	}

	if overridden["proxy/mailpit/enabled"] != "false" {
		t.Errorf("the override did not reach the defaults: %q", overridden["proxy/mailpit/enabled"])
	}

	// And the user has the last word. This mirrors what GetOriginalGeneralConfig
	// does with config.xml: the file is overlaid after the overrides.
	userConfig := map[string]string{"proxy/mailpit/enabled": "true"}
	GeneralConfigMapping(overridden, userConfig)
	if userConfig["proxy/mailpit/enabled"] != "true" {
		t.Errorf("a user asking for mailpit back was overruled by the edition default: %q",
			userConfig["proxy/mailpit/enabled"])
	}
}
