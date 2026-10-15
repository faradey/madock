package shopware

import "testing"

// "6.10" >= "6.5" is false as a string, so a two-digit Shopware minor got no
// PHP, no database and no search engine at all — every answer was "".
func TestTwoDigitMinorIsLaterThanOneDigit(t *testing.T) {
	got := GetVersions("6.10.0.0")
	want := map[string][2]string{
		"Php":        {got.Php, "8.3"},
		"Db":         {got.Db, "10.11"},
		"OpenSearch": {got.OpenSearch, "2.8.0"},
		"Composer":   {got.Composer, "2"},
		"Redis":      {got.Redis, "7.2"},
		"RabbitMQ":   {got.RabbitMQ, "3.13"},
	}
	for name, pair := range want {
		if pair[0] != pair[1] {
			t.Errorf("6.10.0.0: %s = %q, want %q", name, pair[0], pair[1])
		}
	}
}

func TestCurrentReleasesAreUnchanged(t *testing.T) {
	tests := []struct{ ver, php string }{
		{"6.7.13.1", "8.3"},
		{"6.5.8.0", "8.3"},
		{"6.4.20.2", "8.1"},
		{"6.3.5.0", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := GetPhpVersion(tt.ver); got != tt.php {
			t.Errorf("GetPhpVersion(%q) = %q, want %q", tt.ver, got, tt.php)
		}
	}
}
