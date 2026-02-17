package versions

import (
	"os"
	"testing"

	"github.com/faradey/madock/src/helper/configs"
)

func TestMigratePWAToCustom(t *testing.T) {
	// Create temp XML config with PWA platform
	tmpFile, err := os.CreateTemp("", "pwa-migration-*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <activeScope>default</activeScope>
    <scopes>
        <default>
            <platform>pwa</platform>
            <language>nodejs</language>
            <nodejs>
                <version>18.15.0</version>
            </nodejs>
        </default>
    </scopes>
</config>`

	tmpFile.WriteString(xmlContent)
	tmpFile.Close()

	projectConf := map[string]string{
		"platform": "pwa",
		"language": "nodejs",
	}

	// Run migration
	changed := migratePWAToCustom(tmpPath, projectConf)
	if !changed {
		t.Error("migratePWAToCustom() should return true for PWA platform")
	}

	// Verify config was updated
	// ParseXmlFile returns flat keys from the full XML structure,
	// so scoped values have keys like "scopes/default/platform"
	result := configs.ParseXmlFile(tmpPath)
	if result["scopes/default/platform"] != "custom" {
		t.Errorf("scopes/default/platform = %q, want %q", result["scopes/default/platform"], "custom")
	}
	if result["scopes/default/language"] != "nodejs" {
		t.Errorf("scopes/default/language = %q, want %q", result["scopes/default/language"], "nodejs")
	}
	if result["scopes/default/nodejs/enabled"] != "true" {
		t.Errorf("scopes/default/nodejs/enabled = %q, want %q", result["scopes/default/nodejs/enabled"], "true")
	}
}

func TestMigratePWAToCustomSkipsNonPWA(t *testing.T) {
	tests := []struct {
		name     string
		platform string
	}{
		{"magento2", "magento2"},
		{"custom", "custom"},
		{"shopware", "shopware"},
		{"shopify", "shopify"},
		{"prestashop", "prestashop"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectConf := map[string]string{
				"platform": tt.platform,
			}
			changed := migratePWAToCustom("/nonexistent/path", projectConf)
			if changed {
				t.Errorf("migratePWAToCustom() should return false for platform %q", tt.platform)
			}
		})
	}
}
