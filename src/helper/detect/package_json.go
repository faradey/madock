package detect

import (
	"encoding/json"
	"os"

	"github.com/faradey/madock/v3/src/helper/paths"
)

type PackageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// DetectFromPackageJSON tries to detect Node.js platforms from package.json.
func DetectFromPackageJSON(projectPath string) DetectionResult {
	result := DetectionResult{Detected: false}

	pkgPath := projectPath + "/package.json"
	if !paths.IsFileExist(pkgPath) {
		return result
	}

	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return result
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return result
	}

	if version, ok := lookupDep(&pkg, "@medusajs/medusa"); ok {
		result.Platform = "medusa"
		result.Language = "nodejs"
		result.PlatformVersion = cleanVersion(version)
		result.Detected = true
		result.Source = "@medusajs/medusa"
		return result
	}
	if version, ok := lookupDep(&pkg, "@medusajs/framework"); ok {
		result.Platform = "medusa"
		result.Language = "nodejs"
		result.PlatformVersion = cleanVersion(version)
		result.Detected = true
		result.Source = "@medusajs/framework"
		return result
	}

	// BigCommerce Catalyst — Next.js storefront. Detected via the
	// catalyst-specific deps. The starter publishes packages under
	// @bigcommerce/* and includes a top-level `@thebcms/storefront`
	// or `@catalyst/*` workspace depending on version.
	for _, dep := range []string{
		"@bigcommerce/catalyst-core",
		"@bigcommerce/catalyst-client",
		"@bigcommerce/checkout-sdk",
	} {
		if _, ok := lookupDep(&pkg, dep); ok {
			result.Platform = "bigcommerce"
			result.Language = "nodejs"
			result.PlatformVersion = "catalyst"
			result.Detected = true
			result.Source = dep
			return result
		}
	}
	// BigCommerce Stencil — Cornerstone theme dev. Detected via the
	// `@bigcommerce/stencil-cli` devDependency or the unique
	// `cornerstone` package name.
	if _, ok := lookupDep(&pkg, "@bigcommerce/stencil-cli"); ok {
		result.Platform = "bigcommerce"
		result.Language = "nodejs"
		result.PlatformVersion = "stencil"
		result.Detected = true
		result.Source = "@bigcommerce/stencil-cli"
		return result
	}
	if pkg.Name == "cornerstone" {
		result.Platform = "bigcommerce"
		result.Language = "nodejs"
		result.PlatformVersion = "stencil"
		result.Detected = true
		result.Source = "package.json name=cornerstone"
		return result
	}

	return result
}

func lookupDep(pkg *PackageJSON, name string) (string, bool) {
	if v, ok := pkg.Dependencies[name]; ok {
		return v, true
	}
	if v, ok := pkg.DevDependencies[name]; ok {
		return v, true
	}
	return "", false
}
