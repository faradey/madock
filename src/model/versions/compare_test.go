package versions

import "testing"

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"2.4.10", "2.4.9", 1},
		{"2.4.9", "2.4.10", -1},
		{"2.4.7-p10", "2.4.7-p5", 1},
		{"2.4.7-p5", "2.4.7", 1},
		{"2.4.7", "2.4.7-p1", -1},
		{"2.4.8", "2.4.7-p9", 1},
		{"1.13", "1.9", 1},
		{"6.10", "6.5", 1},
		{"6.5", "6.5.0", 0},
		{"v6.7.13.1", "6.7.13.1", 0},
		{"2.4.9-beta1", "2.4.9", 0},
		{" 2.4.9 ", "2.4.9", 0},
		{"", "2.0.0", -1},
		{"dev-main", "2.0.0", -1},
		{"", "", 0},
	}

	for _, tt := range tests {
		if got := Compare(tt.a, tt.b); got != tt.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestAtLeast(t *testing.T) {
	tests := []struct {
		v, min string
		want   bool
	}{
		{"2.4.10", "2.4.9", true},
		{"2.4.9", "2.4.9", true},
		{"2.4.8-p3", "2.4.9", false},
		{"2.4.7-p10", "2.4.7-p5", true},
		{"8.10", "8.5", true},
		{"", "2.0.0", false},
		{"not-a-version", "0", false},
	}

	for _, tt := range tests {
		if got := AtLeast(tt.v, tt.min); got != tt.want {
			t.Errorf("AtLeast(%q, %q) = %v, want %v", tt.v, tt.min, got, tt.want)
		}
	}
}

// A PHP minor with two digits is the case that string comparison gets wrong:
// "8.10" sorts below "8.5". Not a release that exists yet, which is the point —
// nobody would notice until it does.
func TestXdebugForATwoDigitPhpMinor(t *testing.T) {
	if got := GetXdebugVersion("8.10"); got != "3.5.0" {
		t.Errorf("GetXdebugVersion(\"8.10\") = %q, want \"3.5.0\"", got)
	}
}
