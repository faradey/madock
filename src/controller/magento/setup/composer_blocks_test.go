package setup

import "testing"

// Below 2.4.8 Composer refuses the install (league/flysystem 1.x/2.x has an
// advisory and no fixed release); 2.4.8 and later resolve. The boundary is
// the release that moved to flysystem ^3.0, not a guess.
func TestComposerBlocksEveryReleaseBelow248(t *testing.T) {
	for _, v := range []string{"2.4.7", "2.4.7-p10", "2.4.6-p4", "2.4.5", "2.4.2"} {
		if composerBlocks(v) == "" {
			t.Errorf("%s: no warning, and Composer refuses it", v)
		}
	}
	for _, v := range []string{"2.4.8", "2.4.8-p5", "2.4.9", "2.4.10", ""} {
		if why := composerBlocks(v); why != "" {
			t.Errorf("%s: warned %q, and it installs", v, why)
		}
	}
}
