package preset

import (
	"strings"

	"github.com/faradey/madock/v4/src/model/versions"
)

// Unmatched is the refusal for a --preset that names no preset of the
// platform, with the ones that exist.
//
// setup warned "not found, proceeding with manual configuration" and went on,
// and under --yes manual configuration is the defaults: `--preset magento-247`
// — the example the help itself gave, which matches nothing — installed
// Magento 2.4.9 without a word (found by the Indemorio engine, 2026-10-25). A
// script names a preset to get that preset; anything else is a refusal.
func Unmatched(name, platform string) string {
	var lines []string
	for _, p := range GetPresetsByPlatform(platform) {
		lines = append(lines, "  "+p.Versions.PlatformVersion+"  "+p.Name)
	}
	msg := "No " + platform + " preset matches --preset " + name + "; nothing was set up."
	if len(lines) > 0 {
		msg += " Give a version or a name from these:\n" + strings.Join(lines, "\n")
	}
	return msg
}

// Preset represents a pre-configured setup
type Preset struct {
	Name        string
	Description string
	Platform    string
	Versions    versions.ToolsVersions
}

// Presets are built from the version matrix, not kept beside it.
//
// They used to be a second, hand-written copy of the same answers, and it had
// drifted from the matrix in twelve of twenty-two presets — picking "Magento
// 2.4.6-p4 (LTS)" in the menu gave PHP 8.2 and Elasticsearch, typing 2.4.6-p4
// gave PHP 8.1 and OpenSearch. A preset is now a name, a description and a
// platform version in src/model/versions/matrix/<platform>.xml; its versions
// are whatever the matrix gives that version, and the description is filled in
// from them, so neither can disagree with the matrix again.

// GetPresetsByPlatform returns presets for a specific platform.
func GetPresetsByPlatform(platform string) []Preset {
	var out []Preset
	for _, spec := range versions.Presets(platform) {
		v, ok := versions.FromMatrix(platform, spec.Version)
		if !ok {
			continue
		}
		out = append(out, Preset{
			Name:        spec.Name,
			Description: versions.DescribePreset(spec, v),
			Platform:    platform,
			Versions:    v,
		})
	}
	return out
}

func GetMagentoPresets() []Preset     { return GetPresetsByPlatform("magento2") }
func GetMedusaPresets() []Preset      { return GetPresetsByPlatform("medusa") }
func GetSaleorPresets() []Preset      { return GetPresetsByPlatform("saleor") }
func GetSpreePresets() []Preset       { return GetPresetsByPlatform("spree") }
func GetShopifyPresets() []Preset     { return GetPresetsByPlatform("shopify") }
func GetBigcommercePresets() []Preset { return GetPresetsByPlatform("bigcommerce") }
func GetShopwarePresets() []Preset    { return GetPresetsByPlatform("shopware") }
func GetSyliusPresets() []Preset      { return GetPresetsByPlatform("sylius") }

// CustomPreset represents the "Custom configuration" option
var CustomPreset = Preset{
	Name:        "Custom configuration",
	Description: "Configure each service manually",
	Platform:    "",
}
