package preset

import "strings"

// Find resolves what a person typed after `--preset` to one of the presets a
// platform offers.
//
// It exists because six platforms each carried their own copy of this, and the
// copies had drifted into three different behaviours — silently, because none of
// them fails, they simply fail to match:
//
//   - saleor, sylius and spree matched the preset's **name** only, so
//     `--preset=3.23` found nothing while `--preset=catalyst` did;
//   - bigcommerce and shopify also matched the **version**, and lowercased it
//     before comparing;
//   - the alias tables pointed at different things. saleor's `"latest": "3.23"`
//     is a version looked up by substring against the preset's *name*, while
//     bigcommerce's `"next": "catalyst"` is a version compared to
//     `Versions.PlatformVersion` exactly.
//
// So this is the union of all three rather than a choice between them: every
// mapping that resolved before still resolves, and the two behaviours that were
// missing from four platforms are now everywhere. Being more permissive is the
// safe direction — the failure this replaces is "the preset you named does not
// exist", and nothing here can select a preset the platform did not offer.
//
// The order is deliberate and is the one all six copies already had: an exact
// match beats a substring, and both beat an alias. Without it `--preset=2` on a
// platform offering `2` and `2.5` would depend on map iteration order.
func Find(presets []Preset, aliases map[string]string, name string) *Preset {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return nil
	}

	for i := range presets {
		if strings.ToLower(presets[i].Name) == name ||
			strings.ToLower(presets[i].Versions.PlatformVersion) == name {
			return &presets[i]
		}
	}

	for i := range presets {
		lowerName := strings.ToLower(presets[i].Name)
		lowerVersion := strings.ToLower(presets[i].Versions.PlatformVersion)
		if strings.Contains(lowerName, name) || strings.Contains(lowerVersion, name) {
			return &presets[i]
		}
	}

	alias, ok := aliases[name]
	if !ok {
		return nil
	}

	// An alias names either a version or a name, because the two conventions
	// both exist in the tables this replaces. Version first: it is the exact
	// comparison, and the substring on names is what a wrong guess would land on.
	lowerAlias := strings.ToLower(alias)
	for i := range presets {
		if strings.ToLower(presets[i].Versions.PlatformVersion) == lowerAlias {
			return &presets[i]
		}
	}
	for i := range presets {
		if strings.Contains(strings.ToLower(presets[i].Name), lowerAlias) {
			return &presets[i]
		}
	}

	return nil
}
