package preset

import (
	"strings"
	"testing"
)

// The refusal names the presets that do exist, by the version a --preset
// accepts, so whoever typed the wrong one sees the right one.
func TestAnUnmatchedPresetNamesTheOnesThatExist(t *testing.T) {
	msg := Unmatched("magento-247", "magento2")
	for _, want := range []string{"--preset magento-247", "nothing was set up", "2.4.9", "2.4.7  Magento 2.4.7"} {
		if !strings.Contains(msg, want) {
			t.Errorf("no %q in:\n%s", want, msg)
		}
	}
}
