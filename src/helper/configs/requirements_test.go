package configs

import (
	"strings"
	"testing"
)

func TestAProjectAskingForANewerMadockIsRefused(t *testing.T) {
	saved := versionRequirements
	t.Cleanup(func() { versionRequirements = saved })
	versionRequirements = []versionRequirement{{key: "madock/min_version", edition: "madock", current: "4.3.0"}}

	cases := []struct {
		wanted string
		unmet  bool
	}{
		{"", false},
		{"4.3.0", false},
		{"4.2.9", false},
		{"v4.3", false},
		{"4.3.1", true},
		{"5", true},
		{"4.10.0", true}, // a string comparison would call this older than 4.3.0
		{"latest", true}, // not a version: refused rather than guessed
	}
	for _, c := range cases {
		got := UnmetVersionRequirements(map[string]string{"madock/min_version": c.wanted})
		if (len(got) > 0) != c.unmet {
			t.Errorf("min_version %q: unmet=%v, want %v (%v)", c.wanted, len(got) > 0, c.unmet, got)
		}
	}

	got := UnmetVersionRequirements(map[string]string{"madock/min_version": "4.4.0"})
	if len(got) != 1 || !strings.Contains(got[0], "madock 4.4.0 or newer") || !strings.Contains(got[0], "this is madock 4.3.0") {
		t.Errorf("the refusal should name both versions: %v", got)
	}
}

func TestAnEditionCanAddItsOwnRequirement(t *testing.T) {
	saved := versionRequirements
	t.Cleanup(func() { versionRequirements = saved })
	RegisterVersionRequirement("madock_pro/min_version", "madock-pro", "0.64.8")

	got := UnmetVersionRequirements(map[string]string{"madock_pro/min_version": "0.65.0"})
	if len(got) != 1 || !strings.Contains(got[0], "madock-pro 0.65.0 or newer") {
		t.Errorf("an edition's own key was not checked: %v", got)
	}
}
