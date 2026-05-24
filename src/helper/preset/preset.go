package preset

import (
	"github.com/faradey/madock/v3/src/model/versions"
)

// Preset represents a pre-configured setup
type Preset struct {
	Name        string
	Description string
	Platform    string
	Versions    versions.ToolsVersions
}

var presetProviders = map[string]func() []Preset{}

// RegisterPresets registers a preset provider for a platform.
func RegisterPresets(platform string, fn func() []Preset) {
	presetProviders[platform] = fn
}

// GetMagentoPresets returns available Magento presets
func GetMagentoPresets() []Preset {
	return []Preset{
		{
			Name:        "Magento 2.4.9 (Latest)",
			Description: "Latest stable with PHP 8.5, OpenSearch 3.0, Redis 8.0, MariaDB 11.8",
			Platform:    "magento2",
			Versions: versions.ToolsVersions{
				Platform:        "magento2",
				PlatformVersion: "2.4.9",
				Php:             "8.5",
				Db:              "11.8",
				Composer:        "2",
				SearchEngine:    "OpenSearch",
				OpenSearch:      "3.0.0",
				Redis:           "8.0",
				Valkey:          "9.0.0",
				RabbitMQ:        "4.2",
			},
		},
		{
			Name:        "Magento 2.4.8 (Previous)",
			Description: "Previous stable with PHP 8.4, OpenSearch 2.19, Redis 8.0",
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

// GetMedusaPresets returns available Medusa presets
func GetMedusaPresets() []Preset {
	return []Preset{
		{
			Name:        "Medusa 2.x (Latest)",
			Description: "Latest Medusa.js v2 with Node 22, PostgreSQL 17, Redis 7.4",
			Platform:    "medusa",
			Versions: versions.ToolsVersions{
				Platform:        "medusa",
				PlatformVersion: "2",
				Language:        "nodejs",
				NodeJs:          "22.11.0",
				Yarn:            "4.5.0",
				DbType:          "PostgreSQL",
				Db:              "postgres:17",
				Redis:           "7.4",
				RabbitMQ:        "4.2",
			},
		},
		{
			Name:        "Medusa 2.0 (Stable)",
			Description: "Medusa.js v2 baseline with Node 20, PostgreSQL 16, Redis 7.2",
			Platform:    "medusa",
			Versions: versions.ToolsVersions{
				Platform:        "medusa",
				PlatformVersion: "2.0",
				Language:        "nodejs",
				NodeJs:          "20.18.0",
				Yarn:            "4.5.0",
				DbType:          "PostgreSQL",
				Db:              "postgres:16.4",
				Redis:           "7.2.1",
				RabbitMQ:        "3.13.10",
			},
		},
		{
			Name:        "Medusa 1.x (Legacy)",
			Description: "Medusa.js v1 with Node 18, PostgreSQL 14, Redis 7.0",
			Platform:    "medusa",
			Versions: versions.ToolsVersions{
				Platform:        "medusa",
				PlatformVersion: "1",
				Language:        "nodejs",
				NodeJs:          "18.20.0",
				Yarn:            "1.22.22",
				DbType:          "PostgreSQL",
				Db:              "postgres:14.13",
				Redis:           "7.0",
				RabbitMQ:        "3.12.10",
			},
		},
	}
}

// GetSaleorPresets returns available Saleor presets
func GetSaleorPresets() []Preset {
	return []Preset{
		{
			Name:        "Saleor 3.23 (Latest)",
			Description: "Saleor 3.23.x with Python 3.12, PostgreSQL 15, Redis 7.2",
			Platform:    "saleor",
			Versions: versions.ToolsVersions{
				Platform:        "saleor",
				PlatformVersion: "3.23",
				Language:        "python",
				Python:          "3.12",
				DbType:          "PostgreSQL",
				Db:              "postgres:15",
				Redis:           "7.2.5",
			},
		},
		{
			Name:        "Saleor 3.20 (Stable)",
			Description: "Saleor 3.20.x with Python 3.12, PostgreSQL 15, Redis 7.0",
			Platform:    "saleor",
			Versions: versions.ToolsVersions{
				Platform:        "saleor",
				PlatformVersion: "3.20",
				Language:        "python",
				Python:          "3.12",
				DbType:          "PostgreSQL",
				Db:              "postgres:15",
				Redis:           "7.0",
			},
		},
	}
}

// GetSpreePresets returns available Spree presets
func GetSpreePresets() []Preset {
	return []Preset{
		{
			Name:        "Spree 5.x (Latest)",
			Description: "Spree 5.x with Ruby 4.0, PostgreSQL 16, Redis 7.2 (Rails 8)",
			Platform:    "spree",
			Versions: versions.ToolsVersions{
				Platform:        "spree",
				PlatformVersion: "5.0",
				Language:        "ruby",
				Ruby:            "4.0.5",
				DbType:          "PostgreSQL",
				Db:              "postgres:16.4",
				Redis:           "7.2.5",
			},
		},
		{
			Name:        "Spree 4.x (Stable)",
			Description: "Spree 4.10.x with Ruby 3.2, PostgreSQL 15, Redis 7.0 (Rails 7.1)",
			Platform:    "spree",
			Versions: versions.ToolsVersions{
				Platform:        "spree",
				PlatformVersion: "4.10",
				Language:        "ruby",
				Ruby:            "3.2.6",
				DbType:          "PostgreSQL",
				Db:              "postgres:15",
				Redis:           "7.0",
			},
		},
	}
}

// GetShopifyPresets returns available Shopify SDK / framework presets.
// Each preset wires a different stack flavour:
//   - hydrogen        : Node + Hydrogen storefront (Remix on Vite)
//   - app-remix       : Node + Shopify App template (Remix + Prisma)
//   - api-php         : PHP + official shopify-api Composer SDK
//   - laravel-shopify : PHP + Laravel + Kyon147/laravel-shopify package
func GetShopifyPresets() []Preset {
	return []Preset{
		{
			Name:        "Hydrogen (Node.js storefront)",
			Description: "Official headless storefront — Remix on Vite, Node 22, TypeScript. Deploys to Shopify Oxygen.",
			Platform:    "shopify",
			Versions: versions.ToolsVersions{
				Platform:        "shopify",
				PlatformVersion: "hydrogen",
				Language:        "nodejs",
				NodeJs:          "22.20.0",
				Yarn:            "1.22.22",
			},
		},
		{
			Name:        "Shopify App (Node.js / Remix)",
			Description: "Official embedded app template — Remix + Prisma + App Bridge, Node 22, TypeScript.",
			Platform:    "shopify",
			Versions: versions.ToolsVersions{
				Platform:        "shopify",
				PlatformVersion: "app-remix",
				Language:        "nodejs",
				NodeJs:          "22.20.0",
				Yarn:            "1.22.22",
			},
		},
		{
			Name:        "PHP API SDK (shopify-api-php)",
			Description: "Backend integration via official shopify/shopify-api Composer package. PHP 8.3, MariaDB, Redis.",
			Platform:    "shopify",
			Versions: versions.ToolsVersions{
				Platform:        "shopify",
				PlatformVersion: "api-php",
				Language:        "php",
				Php:             "8.3",
				Db:              "mariadb:11.4",
				Composer:        "2",
				Redis:           "7.4",
			},
		},
		{
			Name:        "Laravel Shopify App (Kyon147/laravel-shopify)",
			Description: "Full embedded Shopify App on Laravel — OAuth, billing, webhooks, AppBridge wired. PHP 8.3, MariaDB, Redis, Node for assets.",
			Platform:    "shopify",
			Versions: versions.ToolsVersions{
				Platform:        "shopify",
				PlatformVersion: "laravel-shopify",
				Language:        "php",
				Php:             "8.3",
				Db:              "mariadb:11.4",
				Composer:        "2",
				Redis:           "7.4",
				NodeJs:          "22.20.0",
				Yarn:            "1.22.22",
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

// GetSyliusPresets returns available Sylius presets
func GetSyliusPresets() []Preset {
	return []Preset{
		{
			Name:        "Sylius 2.0 (Latest)",
			Description: "Sylius 2.0.x with PHP 8.3, MariaDB 11.4, Redis 7.4, Node 22, Yarn",
			Platform:    "sylius",
			Versions: versions.ToolsVersions{
				Platform:        "sylius",
				PlatformVersion: "2.0",
				Language:        "php",
				Php:             "8.3",
				Db:              "mariadb:11.4",
				Composer:        "2",
				Redis:           "7.4",
				NodeJs:          "22.20.0",
				Yarn:            "1.22.22",
			},
		},
		{
			Name:        "Sylius 1.13 (Stable)",
			Description: "Sylius 1.13.x with PHP 8.2, MariaDB 10.11, Redis 7.2, Node 20",
			Platform:    "sylius",
			Versions: versions.ToolsVersions{
				Platform:        "sylius",
				PlatformVersion: "1.13",
				Language:        "php",
				Php:             "8.2",
				Db:              "mariadb:10.11",
				Composer:        "2",
				Redis:           "7.2",
				NodeJs:          "20.16.0",
				Yarn:            "1.22.22",
			},
		},
	}
}

func init() {
	RegisterPresets("magento2", GetMagentoPresets)
	RegisterPresets("shopware", GetShopwarePresets)
	RegisterPresets("medusa", GetMedusaPresets)
	RegisterPresets("saleor", GetSaleorPresets)
	RegisterPresets("spree", GetSpreePresets)
	RegisterPresets("sylius", GetSyliusPresets)
	RegisterPresets("shopify", GetShopifyPresets)
}

// GetPresetsByPlatform returns presets for a specific platform.
func GetPresetsByPlatform(platform string) []Preset {
	if fn, ok := presetProviders[platform]; ok {
		return fn()
	}
	return nil
}

// CustomPreset represents the "Custom configuration" option
var CustomPreset = Preset{
	Name:        "Custom configuration",
	Description: "Configure each service manually",
	Platform:    "",
}
