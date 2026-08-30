package preset

import (
	"testing"

	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/model/versions"
)

// What the chooser must get right is the bookkeeping around the question, not
// the question. Six platforms each had this block and each got it right, which
// is exactly why it is worth pinning before it becomes one: the custom entry is
// appended after the real presets, so every index test depends on where it
// sits, and "the person chose custom" and "the person chose the last preset"
// are one off from each other.
func TestChoose(t *testing.T) {
	presets := []Preset{
		{Name: "Catalyst", Description: "storefront", Versions: versions.ToolsVersions{PlatformVersion: "catalyst"}},
		{Name: "Stencil", Description: "theme", Versions: versions.ToolsVersions{PlatformVersion: "stencil"}},
	}

	t.Run("the first preset", func(t *testing.T) {
		answer(t, 0)

		got := Choose(presets, "Choose:")
		if got == nil || got.Name != "Catalyst" {
			t.Errorf("got %v, want Catalyst", got)
		}
	})

	t.Run("the last real preset, which sits one before custom", func(t *testing.T) {
		answer(t, 1)

		got := Choose(presets, "Choose:")
		if got == nil || got.Name != "Stencil" {
			t.Errorf("got %v, want Stencil", got)
		}
	})

	t.Run("custom means no preset", func(t *testing.T) {
		// The index of the appended entry. Returning presets[len-1] here is the
		// off-by-one this test exists for: the person asked to configure by
		// hand and would silently get the last preset instead.
		answer(t, len(presets))

		if got := Choose(presets, "Choose:"); got != nil {
			t.Errorf("got %v, want nothing — custom was chosen", got)
		}
	})

	t.Run("an answer outside the list means no preset", func(t *testing.T) {
		// A selector that was interrupted, or that grows a way of saying "none".
		// Reading past the slice would panic in front of the person setting up.
		for _, idx := range []int{-1, len(presets) + 5} {
			answer(t, idx)

			if got := Choose(presets, "Choose:"); got != nil {
				t.Errorf("index %d gave %v, want nothing", idx, got)
			}
		}
	})

	t.Run("custom is offered, and offered last", func(t *testing.T) {
		var offered []fmtc.PresetOption
		restore := stub(t, func(_ string, options []fmtc.PresetOption) int {
			offered = options
			return 0
		})
		defer restore()

		Choose(presets, "Choose:")

		if len(offered) != len(presets)+1 {
			t.Fatalf("offered %d options for %d presets", len(offered), len(presets))
		}
		if !offered[len(offered)-1].IsCustom {
			t.Error("the custom entry is not last, and every index above depends on that")
		}
		for i := range presets {
			if offered[i].IsCustom {
				t.Errorf("option %d is marked custom and is a real preset", i)
			}
		}
	})
}

// The returned pointer addresses the caller's slice, not a copy of it.
func TestChooseReturnsThePresetItWasGiven(t *testing.T) {
	presets := []Preset{{Name: "Only", Versions: versions.ToolsVersions{PlatformVersion: "1"}}}
	answer(t, 0)

	if got := Choose(presets, "Choose:"); got != &presets[0] {
		t.Error("Choose returned a copy rather than the element it was given")
	}
}

func answer(t *testing.T, index int) {
	t.Helper()

	restore := stub(t, func(string, []fmtc.PresetOption) int { return index })
	t.Cleanup(restore)
}

func stub(t *testing.T, fn func(string, []fmtc.PresetOption) int) func() {
	t.Helper()

	original := selectPreset
	selectPreset = fn

	return func() { selectPreset = original }
}
