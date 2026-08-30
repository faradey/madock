package preset

import (
	"testing"

	"github.com/faradey/madock/v4/src/model/versions"
)

// The cases here are the differences between the six copies this replaces, not
// invented ones. Each of the first three was a real behaviour that existed on
// two platforms and not on the other four, and none of them failed loudly: an
// unmatched preset is reported as "not found", so a platform simply refused a
// name its neighbour accepted.
func TestFindResolvesEveryConventionTheCopiesHad(t *testing.T) {
	presets := []Preset{
		{Name: "Catalyst storefront", Versions: versions.ToolsVersions{PlatformVersion: "catalyst"}},
		{Name: "Stencil theme", Versions: versions.ToolsVersions{PlatformVersion: "stencil"}},
		{Name: "Saleor 3.23", Versions: versions.ToolsVersions{PlatformVersion: "3.23"}},
		{Name: "Saleor 3.20", Versions: versions.ToolsVersions{PlatformVersion: "3.20"}},
	}
	aliases := map[string]string{
		"latest": "3.23",     // a version, the saleor convention
		"next":   "catalyst", // a version, the bigcommerce convention
		"theme":  "Stencil",  // a name, matched by substring
	}

	cases := []struct {
		name string
		want string
		why  string
	}{
		{"catalyst", "Catalyst storefront", "exact match on the version — four platforms could not do this"},
		{"Catalyst storefront", "Catalyst storefront", "exact match on the name, which every copy had"},
		{"CATALYST", "Catalyst storefront", "case is not the user's problem"},
		{"3.23", "Saleor 3.23", "a version typed directly, which is the commonest way to ask"},
		{"stenc", "Stencil theme", "substring on the name"},
		{"3.2", "Saleor 3.23", "substring on the version, and the first offered wins"},
		{"latest", "Saleor 3.23", "alias pointing at a version"},
		{"next", "Catalyst storefront", "alias pointing at a version, the other table's convention"},
		{"theme", "Stencil theme", "alias pointing at a name, resolved by substring"},
	}

	for _, c := range cases {
		got := Find(presets, aliases, c.name)
		if got == nil {
			t.Errorf("Find(%q) found nothing — %s", c.name, c.why)
			continue
		}
		if got.Name != c.want {
			t.Errorf("Find(%q) = %q, want %q — %s", c.name, got.Name, c.want, c.why)
		}
	}
}

// What must not resolve. A preset selected by accident is worse than one not
// found: the run continues, and it continues with the wrong versions.
func TestFindRefusesWhatItCannotResolve(t *testing.T) {
	presets := []Preset{
		{Name: "Catalyst storefront", Versions: versions.ToolsVersions{PlatformVersion: "catalyst"}},
	}
	aliases := map[string]string{"next": "nothing-offers-this"}

	for _, name := range []string{"", "   ", "hydrogen", "next"} {
		if got := Find(presets, aliases, name); got != nil {
			t.Errorf("Find(%q) = %q, want nothing", name, got.Name)
		}
	}
}

// Exact beats substring, and both beat an alias.
//
// Not a stylistic preference: with `2` and `2.5` both offered, a substring rule
// applied first would answer whichever the slice happened to hold first, and the
// person asking for `2` would get 2.5 on a good day. All six copies had this
// order already; it is written down here so a rewrite cannot lose it quietly.
func TestFindPrefersTheExactAnswer(t *testing.T) {
	presets := []Preset{
		{Name: "Two point five", Versions: versions.ToolsVersions{PlatformVersion: "2.5"}},
		{Name: "Two", Versions: versions.ToolsVersions{PlatformVersion: "2"}},
	}
	aliases := map[string]string{"2": "2.5"}

	got := Find(presets, aliases, "2")
	if got == nil || got.Name != "Two" {
		t.Errorf("Find(2) = %v, want the preset whose version is exactly 2", got)
	}
}

// The returned pointer addresses the caller's slice, not a loop copy.
//
// Every copy this replaces wrote `for _, p := range presets { return &p }`,
// which was a bug in Go before 1.22 and is merely fragile now. A caller that
// changes what it is handed should see it in what it got back.
func TestFindReturnsThePresetItWasGiven(t *testing.T) {
	presets := []Preset{
		{Name: "Catalyst", Versions: versions.ToolsVersions{PlatformVersion: "catalyst"}},
	}

	got := Find(presets, nil, "catalyst")
	if got != &presets[0] {
		t.Error("Find returned a copy rather than the element of the slice it was given")
	}
}
