package detect

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"github.com/faradey/madock/src/helper/paths"
)

// ComposerJSON represents the structure of composer.json
type ComposerJSON struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Require map[string]string `json:"require"`
	Version string            `json:"version"`
}

// DetectionResult contains the detected platform information
type DetectionResult struct {
	Platform        string
	PlatformVersion string
	Detected        bool
	Source          string
}

// DetectFromComposer tries to detect platform and version from composer.json
func DetectFromComposer(projectPath string) DetectionResult {
	result := DetectionResult{
		Detected: false,
	}

	composerPath := projectPath + "/composer.json"
	if !paths.IsFileExist(composerPath) {
		return result
	}

	data, err := os.ReadFile(composerPath)
	if err != nil {
		return result
	}

	var composer ComposerJSON
	if err := json.Unmarshal(data, &composer); err != nil {
		return result
	}

	// Check for Magento 2
	if version, ok := composer.Require["magento/product-community-edition"]; ok {
		result.Platform = "magento2"
		result.PlatformVersion = cleanVersion(version)
		result.Detected = true
		result.Source = "magento/product-community-edition"
		return result
	}

	if version, ok := composer.Require["magento/product-enterprise-edition"]; ok {
		result.Platform = "magento2"
		result.PlatformVersion = cleanVersion(version)
		result.Detected = true
		result.Source = "magento/product-enterprise-edition"
		return result
	}

	// Check for magento/magento2-base (alternative detection)
	if version, ok := composer.Require["magento/magento2-base"]; ok {
		result.Platform = "magento2"
		result.PlatformVersion = cleanVersion(version)
		result.Detected = true
		result.Source = "magento/magento2-base"
		return result
	}

	// Check for Shopware
	if version, ok := composer.Require["shopware/core"]; ok {
		result.Platform = "shopware"
		result.PlatformVersion = cleanVersion(version)
		result.Detected = true
		result.Source = "shopware/core"
		return result
	}

	// Check for PrestaShop
	if version, ok := composer.Require["prestashop/prestashop"]; ok {
		result.Platform = "prestashop"
		result.PlatformVersion = cleanVersion(version)
		result.Detected = true
		result.Source = "prestashop/prestashop"
		return result
	}

	// Check project type
	if composer.Type == "magento2-module" || composer.Type == "magento2-theme" {
		result.Platform = "magento2"
		result.Detected = true
		result.Source = "project type"
		return result
	}

	return result
}

// DetectFromComposerLock tries to detect from composer.lock for more accurate version
func DetectFromComposerLock(projectPath string) DetectionResult {
	result := DetectionResult{
		Detected: false,
	}

	lockPath := projectPath + "/composer.lock"
	if !paths.IsFileExist(lockPath) {
		return result
	}

	data, err := os.ReadFile(lockPath)
	if err != nil {
		return result
	}

	var lockFile struct {
		Packages []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"packages"`
	}

	if err := json.Unmarshal(data, &lockFile); err != nil {
		return result
	}

	for _, pkg := range lockFile.Packages {
		switch pkg.Name {
		case "magento/product-community-edition", "magento/product-enterprise-edition":
			result.Platform = "magento2"
			result.PlatformVersion = cleanVersion(pkg.Version)
			result.Detected = true
			result.Source = pkg.Name + " (from lock)"
			return result
		case "shopware/core":
			result.Platform = "shopware"
			result.PlatformVersion = cleanVersion(pkg.Version)
			result.Detected = true
			result.Source = pkg.Name + " (from lock)"
			return result
		}
	}

	return result
}

// Detect tries all detection methods
func Detect(projectPath string) DetectionResult {
	// Try composer.lock first (more accurate)
	if result := DetectFromComposerLock(projectPath); result.Detected {
		return result
	}

	// Fall back to composer.json
	if result := DetectFromComposer(projectPath); result.Detected {
		return result
	}

	return DetectionResult{Detected: false}
}

// cleanVersion removes version constraints and returns clean version
func cleanVersion(version string) string {
	// Remove common version prefixes/constraints
	version = strings.TrimPrefix(version, "^")
	version = strings.TrimPrefix(version, "~")
	version = strings.TrimPrefix(version, ">=")
	version = strings.TrimPrefix(version, ">")
	version = strings.TrimPrefix(version, "<=")
	version = strings.TrimPrefix(version, "<")
	version = strings.TrimPrefix(version, "=")
	version = strings.TrimPrefix(version, "v")

	// Handle ranges like "2.4.6 - 2.4.7" - take the first version
	if strings.Contains(version, " - ") {
		parts := strings.Split(version, " - ")
		version = strings.TrimSpace(parts[0])
	}

	// Handle OR conditions like "2.4.6 || 2.4.7" - take the first
	if strings.Contains(version, "||") {
		parts := strings.Split(version, "||")
		version = strings.TrimSpace(parts[0])
	}

	// Handle AND conditions
	if strings.Contains(version, " ") {
		parts := strings.Fields(version)
		version = parts[0]
	}

	// Remove any remaining constraint characters
	re := regexp.MustCompile(`[^0-9.]`)
	version = re.ReplaceAllString(version, "")

	// Trim trailing dots
	version = strings.TrimSuffix(version, ".")

	return version
}
