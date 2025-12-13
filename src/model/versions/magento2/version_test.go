package magento2

import "testing"

func TestGetPhpVersion(t *testing.T) {
	tests := []struct {
		mageVer  string
		expected string
	}{
		// Latest versions
		{"2.4.8", "8.4"},
		{"2.4.8-p1", "8.4"},
		{"2.4.7", "8.3"},
		{"2.4.7-p5", "8.3"},
		// PHP 8.1 range
		{"2.4.6", "8.1"},
		{"2.4.5", "8.1"},
		{"2.4.4", "8.1"},
		// PHP 7.4 range
		{"2.3.7", "7.4"},
		{"2.3.7-p4", "7.4"},
		// PHP 7.3 range
		{"2.3.6", "7.3"},
		{"2.3.5", "7.3"},
		{"2.3.4", "7.3"},
		{"2.3.3", "7.3"},
		// PHP 7.2 range
		{"2.3.2", "7.2"},
		{"2.3.1", "7.2"},
		{"2.3.0", "7.2"},
		// PHP 7.1 range
		{"2.2.9", "7.1"},
		{"2.2.0", "7.1"},
		// PHP 7.0 range
		{"2.1.0", "7.0"},
		{"2.0.0", "7.0"},
		// Unknown/empty
		{"1.9.0", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.mageVer, func(t *testing.T) {
			got := GetPhpVersion(tt.mageVer)
			if got != tt.expected {
				t.Errorf("GetPhpVersion(%q) = %q, want %q", tt.mageVer, got, tt.expected)
			}
		})
	}
}

func TestGetDBVersion(t *testing.T) {
	tests := []struct {
		mageVer  string
		expected string
	}{
		{"2.4.8", "11.4"},
		{"2.4.7", "10.6"},
		{"2.4.6", "10.4"},
		{"2.4.5", "10.4"},
		{"2.4.4", "10.4"},
		{"2.4.3", "10.4"},
		{"2.4.2", "10.4"},
		{"2.4.1", "10.4"},
		{"2.4.0", "10.3"},
		{"2.3.7", "10.3"},
		{"2.3.6", "10.2"},
		{"2.3.5", "10.2"},
		{"2.3.4", "10.2"},
		{"2.3.3", "10.2"},
		{"2.3.2", "10.2"},
		{"2.3.1", "10.2"},
		{"2.3.0", "10.2"},
		{"2.2.0", "10.0"},
		{"2.0.0", "10.0"},
		{"1.9.0", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.mageVer, func(t *testing.T) {
			got := GetDBVersion(tt.mageVer)
			if got != tt.expected {
				t.Errorf("GetDBVersion(%q) = %q, want %q", tt.mageVer, got, tt.expected)
			}
		})
	}
}

func TestGetSearchEngineVersion(t *testing.T) {
	tests := []struct {
		mageVer  string
		expected string
	}{
		{"2.4.9", "OpenSearch"},
		{"2.4.8", "OpenSearch"},
		{"2.4.7", "OpenSearch"},
		{"2.4.6", "OpenSearch"},
		{"2.4.5", "Elasticsearch"},
		{"2.4.4", "Elasticsearch"},
		{"2.4.3", "Elasticsearch"},
		{"2.3.7", "Elasticsearch"},
		{"2.0.0", "Elasticsearch"},
	}

	for _, tt := range tests {
		t.Run(tt.mageVer, func(t *testing.T) {
			got := GetSearchEngineVersion(tt.mageVer)
			if got != tt.expected {
				t.Errorf("GetSearchEngineVersion(%q) = %q, want %q", tt.mageVer, got, tt.expected)
			}
		})
	}
}

func TestGetOpenSearchVersion(t *testing.T) {
	tests := []struct {
		mageVer  string
		expected string
	}{
		{"2.4.9", "3.0.0"},
		{"2.4.8", "2.19.0"},
		{"2.4.7", "2.12.0"},
		{"2.4.6", "2.5.0"},
		// Note: 2.4.5 and 2.4.4 return "1.2.0" due to string comparison (>= "2.4.3-p2")
		{"2.4.5", "1.2.0"},
		{"2.4.4", "1.2.0"},
		{"2.4.3-p2", "1.2.0"},
		{"2.4.3-p3", "1.2.0"},
		{"2.3.7-p4", "1.2.0"},
		{"2.3.7-p3", "1.2.0"},
		{"2.3.7", "NotCompatible"},
		{"2.4.3", "NotCompatible"},
	}

	for _, tt := range tests {
		t.Run(tt.mageVer, func(t *testing.T) {
			got := GetOpenSearchVersion(tt.mageVer)
			if got != tt.expected {
				t.Errorf("GetOpenSearchVersion(%q) = %q, want %q", tt.mageVer, got, tt.expected)
			}
		})
	}
}

func TestGetElasticVersion(t *testing.T) {
	tests := []struct {
		mageVer  string
		expected string
	}{
		{"2.4.8", "8.17.6"},
		{"2.4.7", "8.11.14"},
		{"2.4.6", "8.4.3"},
		{"2.4.5", "7.17.5"},
		{"2.4.4", "7.16.3"},
		{"2.4.3", "7.10.1"},
		{"2.4.2", "7.9.3"},
		{"2.4.1", "7.7.1"},
		{"2.4.0", "7.6.2"},
		{"2.3.7", "7.9.3"},
		{"2.3.6", "7.7.1"},
		{"2.3.5", "7.6.2"},
		{"2.3.4", "6.8.13"},
		{"2.3.1", "6.8.13"},
		{"2.0.0", "6.8.13"},
		{"1.9.0", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.mageVer, func(t *testing.T) {
			got := GetElasticVersion(tt.mageVer)
			if got != tt.expected {
				t.Errorf("GetElasticVersion(%q) = %q, want %q", tt.mageVer, got, tt.expected)
			}
		})
	}
}

func TestGetComposerVersion(t *testing.T) {
	tests := []struct {
		mageVer  string
		expected string
	}{
		{"2.4.8", "2"},
		{"2.4.7", "2"},
		{"2.4.6", "2"},
		{"2.4.2", "2"},
		{"2.4.1", "1"},
		{"2.4.0", "1"},
		{"2.3.7", "2"},
		{"2.3.6", "1"},
		{"2.3.0", "1"},
		{"2.0.0", "1"},
		{"1.9.0", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.mageVer, func(t *testing.T) {
			got := GetComposerVersion(tt.mageVer)
			if got != tt.expected {
				t.Errorf("GetComposerVersion(%q) = %q, want %q", tt.mageVer, got, tt.expected)
			}
		})
	}
}

func TestGetRedisVersion(t *testing.T) {
	tests := []struct {
		mageVer  string
		expected string
	}{
		{"2.4.8", "8.0"},
		{"2.4.7", "7.2"},
		{"2.4.6", "7.0"},
		{"2.4.5", "6.2"},
		{"2.4.4", "6.2"},
		{"2.4.3", "6.0"},
		{"2.4.2", "6.0"},
		{"2.4.1", "5.0"},
		{"2.4.0", "5.0"},
		{"2.3.7", "6.0"},
		{"2.3.6", "5.0"},
		{"2.0.0", "5.0"},
		{"1.9.0", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.mageVer, func(t *testing.T) {
			got := GetRedisVersion(tt.mageVer)
			if got != tt.expected {
				t.Errorf("GetRedisVersion(%q) = %q, want %q", tt.mageVer, got, tt.expected)
			}
		})
	}
}

func TestGetRabbitMQVersion(t *testing.T) {
	tests := []struct {
		mageVer  string
		expected string
	}{
		{"2.4.8", "4.1"},
		{"2.4.7-p5", "4.1"},
		{"2.4.7-p4", "3.13"},
		{"2.4.7", "3.13"},
		{"2.4.6", "3.11"},
		{"2.4.5", "3.9"},
		{"2.4.4", "3.9"},
		{"2.4.3", "3.8"},
		{"2.3.7", "3.8"},
		{"2.3.4", "3.8"},
		{"2.3.3", "3.7"},
		{"2.0.0", "3.7"},
		{"1.9.0", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.mageVer, func(t *testing.T) {
			got := GetRabbitMQVersion(tt.mageVer)
			if got != tt.expected {
				t.Errorf("GetRabbitMQVersion(%q) = %q, want %q", tt.mageVer, got, tt.expected)
			}
		})
	}
}

func TestGetValkeyVersion(t *testing.T) {
	tests := []struct {
		mageVer  string
		expected string
	}{
		{"2.4.8", "8.1.3"},
		{"2.4.7", "8.1.3"},
		{"2.0.0", "8.1.3"},
		{"", "8.1.3"},
	}

	for _, tt := range tests {
		t.Run(tt.mageVer, func(t *testing.T) {
			got := GetValkeyVersion(tt.mageVer)
			if got != tt.expected {
				t.Errorf("GetValkeyVersion(%q) = %q, want %q", tt.mageVer, got, tt.expected)
			}
		})
	}
}

func TestGetVersions(t *testing.T) {
	// Test that GetVersions returns a complete struct for known version
	versions := GetVersions("2.4.7")

	if versions.Platform != "magento2" {
		t.Errorf("Expected platform magento2, got %s", versions.Platform)
	}
	if versions.Php != "8.3" {
		t.Errorf("Expected PHP 8.3, got %s", versions.Php)
	}
	if versions.Db != "10.6" {
		t.Errorf("Expected DB 10.6, got %s", versions.Db)
	}
	if versions.SearchEngine != "OpenSearch" {
		t.Errorf("Expected SearchEngine OpenSearch, got %s", versions.SearchEngine)
	}
	if versions.OpenSearch != "2.12.0" {
		t.Errorf("Expected OpenSearch 2.12.0, got %s", versions.OpenSearch)
	}
	if versions.PlatformVersion != "2.4.7" {
		t.Errorf("Expected PlatformVersion 2.4.7, got %s", versions.PlatformVersion)
	}
}
