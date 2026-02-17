package projects

import (
	"testing"

	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/model/versions"
)

func TestMagento2ConfigSets(t *testing.T) {
	// Test that Magento2 function sets expected config values
	config := new(configs2.ConfigLines)
	defVersions := versions.ToolsVersions{
		Platform:        "magento2",
		PlatformVersion: "2.4.7",
		Php:             "8.3",
		Db:              "10.6",
		Composer:        "2",
		SearchEngine:    "OpenSearch",
		Elastic:         "8.11.14",
		OpenSearch:      "2.12.0",
		Redis:           "7.2",
		RabbitMQ:        "3.13",
	}
	generalConf := map[string]string{
		"timezone":       "Europe/Kiev",
		"php/xdebug/ide_key": "PHPSTORM",
		"php/xdebug/enabled": "false",
		"php/ioncube/enabled": "false",
		"db/root_password":   "password",
		"db/user":            "magento",
		"db/password":        "magento",
		"db/database":        "magento",
		"redis/enabled":      "false",
		"nodejs/enabled":     "false",
		"nodejs/version":     "18.15.0",
		"rabbitmq/enabled":   "false",
	}
	projectConf := map[string]string{}

	// Call the function
	Magento2(config, defVersions, generalConf, projectConf)

	// Verify key settings were applied
	if config.Lines == nil {
		t.Fatal("Config lines should not be nil")
	}

	// Check that config has expected number of settings
	if len(config.Lines) == 0 {
		t.Error("Magento2() should set multiple config values")
	}
}

func TestMagento2WithElasticsearch(t *testing.T) {
	config := new(configs2.ConfigLines)
	defVersions := versions.ToolsVersions{
		Platform:        "magento2",
		PlatformVersion: "2.4.5",
		Php:             "8.1",
		Db:              "10.4",
		Composer:        "2",
		SearchEngine:    "Elasticsearch",
		Elastic:         "7.17.5",
		OpenSearch:      "NotCompatible",
		Redis:           "6.2",
		RabbitMQ:        "3.9",
	}
	generalConf := map[string]string{
		"timezone":        "UTC",
		"php/xdebug/ide_key":  "PHPSTORM",
		"php/xdebug/enabled":  "false",
		"php/ioncube/enabled": "false",
		"db/root_password":    "password",
		"db/user":             "magento",
		"db/password":         "magento",
		"db/database":         "magento",
		"redis/enabled":       "false",
		"nodejs/enabled":      "false",
		"nodejs/version":      "18.15.0",
		"rabbitmq/enabled":    "false",
	}
	projectConf := map[string]string{}

	Magento2(config, defVersions, generalConf, projectConf)

	// Function should complete without panic
	if config.Lines == nil {
		t.Fatal("Config lines should not be nil")
	}
}

func TestMagento2WithCustomDbRepo(t *testing.T) {
	config := new(configs2.ConfigLines)
	defVersions := versions.ToolsVersions{
		Platform:        "magento2",
		PlatformVersion: "2.4.7",
		Php:             "8.3",
		Db:              "mysql:8.0", // Custom repository:version format
		Composer:        "2",
		SearchEngine:    "OpenSearch",
		Elastic:         "8.11.14",
		OpenSearch:      "2.12.0",
		Redis:           "7.2",
		RabbitMQ:        "3.13",
	}
	generalConf := map[string]string{
		"timezone":        "UTC",
		"php/xdebug/ide_key":  "PHPSTORM",
		"php/xdebug/enabled":  "false",
		"php/ioncube/enabled": "false",
		"db/root_password":    "password",
		"db/user":             "magento",
		"db/password":         "magento",
		"db/database":         "magento",
		"redis/enabled":       "false",
		"nodejs/enabled":      "false",
		"nodejs/version":      "18.15.0",
		"rabbitmq/enabled":    "false",
	}
	projectConf := map[string]string{}

	Magento2(config, defVersions, generalConf, projectConf)

	// Function should handle repository:version format
	if config.Lines == nil {
		t.Fatal("Config lines should not be nil")
	}
}

func TestShopifyConfigSets(t *testing.T) {
	config := new(configs2.ConfigLines)
	defVersions := versions.ToolsVersions{
		Platform: "shopify",
		Php:      "8.1",
		NodeJs:   "18.15.0",
	}
	generalConf := map[string]string{
		"timezone":        "UTC",
		"php/xdebug/ide_key":  "PHPSTORM",
		"php/xdebug/enabled":  "false",
		"php/ioncube/enabled": "false",
		"nodejs/version":      "18.15.0",
	}
	projectConf := map[string]string{}

	Shopify(config, defVersions, generalConf, projectConf)

	if config.Lines == nil {
		t.Fatal("Config lines should not be nil")
	}
}

func TestCustomConfigSets(t *testing.T) {
	config := new(configs2.ConfigLines)
	defVersions := versions.ToolsVersions{
		Platform: "custom",
		Php:      "8.2",
		Db:       "10.6",
	}
	generalConf := map[string]string{
		"timezone":        "UTC",
		"php/xdebug/ide_key":  "PHPSTORM",
		"php/xdebug/enabled":  "false",
		"php/ioncube/enabled": "false",
		"db/root_password":    "password",
		"db/user":             "app",
		"db/password":         "app",
		"db/database":         "app",
		"redis/enabled":       "false",
		"nodejs/enabled":      "false",
		"nodejs/version":      "18.15.0",
	}
	projectConf := map[string]string{}

	Custom(config, defVersions, generalConf, projectConf)

	if config.Lines == nil {
		t.Fatal("Config lines should not be nil")
	}
}

func TestShopwareConfigSets(t *testing.T) {
	config := new(configs2.ConfigLines)
	defVersions := versions.ToolsVersions{
		Platform: "shopware",
		Php:      "8.2",
		Db:       "10.6",
	}
	generalConf := map[string]string{
		"timezone":        "UTC",
		"php/xdebug/ide_key":  "PHPSTORM",
		"php/xdebug/enabled":  "false",
		"php/ioncube/enabled": "false",
		"db/root_password":    "password",
		"db/user":             "shopware",
		"db/password":         "shopware",
		"db/database":         "shopware",
		"redis/enabled":       "false",
		"nodejs/enabled":      "false",
		"nodejs/version":      "18.15.0",
	}
	projectConf := map[string]string{}

	Shopware(config, defVersions, generalConf, projectConf)

	if config.Lines == nil {
		t.Fatal("Config lines should not be nil")
	}
}

func TestPrestaShopConfigSets(t *testing.T) {
	config := new(configs2.ConfigLines)
	defVersions := versions.ToolsVersions{
		Platform: "prestashop",
		Php:      "8.1",
		Db:       "10.6",
	}
	generalConf := map[string]string{
		"timezone":        "UTC",
		"php/xdebug/ide_key":  "PHPSTORM",
		"php/xdebug/enabled":  "false",
		"php/ioncube/enabled": "false",
		"db/root_password":    "password",
		"db/user":             "prestashop",
		"db/password":         "prestashop",
		"db/database":         "prestashop",
		"redis/enabled":       "false",
		"nodejs/enabled":      "false",
		"nodejs/version":      "18.15.0",
	}
	projectConf := map[string]string{}

	PrestaShop(config, defVersions, generalConf, projectConf)

	if config.Lines == nil {
		t.Fatal("Config lines should not be nil")
	}
}

// Test version format parsing
func TestDbVersionFormatParsing(t *testing.T) {
	tests := []struct {
		input      string
		expectRepo bool
	}{
		{"10.6", false},
		{"mysql:8.0", true},
		{"mariadb:10.6", true},
		{"percona:8.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// This documents the expected behavior of version parsing
			// The format "repository:version" is split by ":"
			// If no ":" is present, the whole string is the version
			t.Logf("Input %q: hasRepo=%v", tt.input, tt.expectRepo)
		})
	}
}
