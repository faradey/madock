package detect

import (
	"testing"

	"github.com/faradey/madock/v4/src/model/versions/magento2"
)

func TestCleanVersionKeepsTheMagentoPatchSuffix(t *testing.T) {
	tests := map[string]string{
		"2.4.7-p3":       "2.4.7-p3",
		"^2.4.7-p3":      "2.4.7-p3",
		"2.4.6-p10":      "2.4.6-p10",
		"2.4.7":          "2.4.7",
		"~2.4.6":         "2.4.6",
		">=2.4.6 <2.5":   "2.4.6",
		"2.4.6 || 2.4.7": "2.4.6",
		"v6.7.13.1":      "6.7.13.1",
		"2.4.8-beta1":    "2.4.8",
		"dev-main":       "",
	}
	for in, want := range tests {
		if got := cleanVersion(in); got != want {
			t.Errorf("cleanVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

// What the suffix decides: a store pinned to a patch release of 2.4.7 needs
// the 2.4.7 stack. Read as "2.4.73" it was later than 2.4.9 and got PHP 8.5.
func TestAPatchReleaseGetsTheStackOfItsLine(t *testing.T) {
	if got := magento2.GetPhpVersion(cleanVersion("2.4.7-p3")); got != "8.3" {
		t.Errorf("PHP for a detected 2.4.7-p3 = %q, want 8.3", got)
	}
	if got := magento2.GetPhpVersion(cleanVersion("2.4.5-p10")); got != "8.1" {
		t.Errorf("PHP for a detected 2.4.5-p10 = %q, want 8.1", got)
	}
}
