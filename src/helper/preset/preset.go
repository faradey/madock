package preset

import (
	"github.com/faradey/madock/src/model/versions"
)

// Preset represents a pre-configured setup
type Preset struct {
	Name        string
	Description string
	Platform    string
	Versions    versions.ToolsVersions
}

// GetMagentoPresets returns available Magento presets
func GetMagentoPresets() []Preset {
	return []Preset{
		{
			Name:        "Magento 2.4.8 (Latest)",
			Description: "Latest stable with PHP 8.4, OpenSearch 2.19, Redis 8.0",
			Platform:    "magento2",
			Versions: versions.ToolsVersions{
				Platform:        "magento2",
				PlatformVersion: "2.4.8",
				Php:             "8.4",
				Db:              "11.4",
				Composer:        "2",
				SearchEngine:    "OpenSearch",
				OpenSearch:      "2.19.0",
				Redis:           "8.0",
				Valkey:          "8.1.3",
				RabbitMQ:        "4.1",
			},
		},
		{
			Name:        "Magento 2.4.8 + Elasticsearch",
			Description: "PHP 8.4, Elasticsearch 8.15, Redis 8.0, RabbitMQ 4.1",
			Platform:    "magento2",
			Versions: versions.ToolsVersions{
				Platform:        "magento2",
				PlatformVersion: "2.4.8",
				Php:             "8.4",
				Db:              "11.4",
				Composer:        "2",
				SearchEngine:    "Elasticsearch",
				Elastic:         "8.15.0",
				Redis:           "8.0",
				RabbitMQ:        "4.1",
			},
		},
		{
			Name:        "Magento 2.4.8 + Valkey",
			Description: "PHP 8.4, OpenSearch 2.19, Valkey 8.1 (no Redis)",
			Platform:    "magento2",
			Versions: versions.ToolsVersions{
				Platform:        "magento2",
				PlatformVersion: "2.4.8",
				Php:             "8.4",
				Db:              "11.4",
				Composer:        "2",
				SearchEngine:    "OpenSearch",
				OpenSearch:      "2.19.0",
				Valkey:          "8.1.3",
				RabbitMQ:        "4.1",
			},
		},
		{
			Name:        "Magento 2.4.8 Minimal",
			Description: "PHP 8.4, OpenSearch 2.19 only (no Redis/Valkey/RabbitMQ)",
			Platform:    "magento2",
			Versions: versions.ToolsVersions{
				Platform:        "magento2",
				PlatformVersion: "2.4.8",
				Php:             "8.4",
				Db:              "11.4",
				Composer:        "2",
				SearchEngine:    "OpenSearch",
				OpenSearch:      "2.19.0",
			},
		},
		{
			Name:        "Magento 2.4.8 + Elasticsearch + Valkey",
			Description: "PHP 8.4, Elasticsearch 8.15, Valkey 8.1, RabbitMQ 4.1",
			Platform:    "magento2",
			Versions: versions.ToolsVersions{
				Platform:        "magento2",
				PlatformVersion: "2.4.8",
				Php:             "8.4",
				Db:              "11.4",
				Composer:        "2",
				SearchEngine:    "Elasticsearch",
				Elastic:         "8.15.0",
				Valkey:          "8.1.3",
				RabbitMQ:        "4.1",
			},
		},
		{
			Name:        "Magento 2.4.7 (Stable)",
			Description: "Stable release with PHP 8.3, OpenSearch 2.12, Redis 7.2",
			Platform:    "magento2",
			Versions: versions.ToolsVersions{
				Platform:        "magento2",
				PlatformVersion: "2.4.7",
				Php:             "8.3",
				Db:              "10.6",
				Composer:        "2",
				SearchEngine:    "OpenSearch",
				OpenSearch:      "2.12.0",
				Redis:           "7.2",
				Valkey:          "8.1.3",
				RabbitMQ:        "3.13",
			},
		},
		{
			Name:        "Magento 2.4.6-p4 (LTS)",
			Description: "Long-term support with PHP 8.2, Elasticsearch 8.11",
			Platform:    "magento2",
			Versions: versions.ToolsVersions{
				Platform:        "magento2",
				PlatformVersion: "2.4.6-p4",
				Php:             "8.2",
				Db:              "10.6",
				Composer:        "2",
				SearchEngine:    "Elasticsearch",
				Elastic:         "8.11.4",
				Redis:           "7.0",
				Valkey:          "8.1.3",
				RabbitMQ:        "3.12",
			},
		},
		{
			Name:        "Magento 2.4.5 (Legacy)",
			Description: "Older stable with PHP 8.1, Elasticsearch 7.17",
			Platform:    "magento2",
			Versions: versions.ToolsVersions{
				Platform:        "magento2",
				PlatformVersion: "2.4.5",
				Php:             "8.1",
				Db:              "10.4",
				Composer:        "2",
				SearchEngine:    "Elasticsearch",
				Elastic:         "7.17.5",
				Redis:           "6.2",
				Valkey:          "8.1.3",
				RabbitMQ:        "3.9",
			},
		},
	}
}

// GetShopwarePresets returns available Shopware presets
func GetShopwarePresets() []Preset {
	return []Preset{
		{
			Name:        "Shopware 6.5 (Latest)",
			Description: "Latest stable with PHP 8.2",
			Platform:    "shopware",
			Versions: versions.ToolsVersions{
				Platform:        "shopware",
				PlatformVersion: "6.5",
				Php:             "8.2",
				Db:              "10.6",
				Composer:        "2",
				Redis:           "7.2",
			},
		},
	}
}

// GetPresetsByPlatform returns presets for a specific platform
func GetPresetsByPlatform(platform string) []Preset {
	switch platform {
	case "magento2":
		return GetMagentoPresets()
	case "shopware":
		return GetShopwarePresets()
	default:
		return nil
	}
}

// CustomPreset represents the "Custom configuration" option
var CustomPreset = Preset{
	Name:        "Custom configuration",
	Description: "Configure each service manually",
	Platform:    "",
}
