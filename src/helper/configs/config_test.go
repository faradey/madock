package configs

import (
	"testing"
)

// ---------------------------------------------------------------------------
// evaluateCondition
// ---------------------------------------------------------------------------

func TestIsSecretKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		// Original keys
		{"db/root_password", true},
		{"db/password", true},
		{"ssh/password", true},
		{"magento/admin_password", true},
		// New service keys
		{"rabbitmq/password", true},
		{"grafana/auth/password", true},
		{"redis/auth/password", true},
		{"valkey/auth/password", true},
		{"search/elasticsearch/auth/password", true},
		{"search/opensearch/auth/password", true},
		// Non-secret keys
		{"rabbitmq/user", false},
		{"grafana/auth/user", false},
		{"rabbitmq/enabled", false},
		{"platform", false},
		{"", false},
		// Scoped variants
		{"scopes/default/rabbitmq/password", true},
		{"scopes/default/grafana/auth/password", true},
		{"scopes/default/search/elasticsearch/auth/password", true},
		{"scopes/staging/redis/auth/password", true},
		{"scopes/default/rabbitmq/user", false},
		{"scopes/default/platform", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := isSecretKey(tt.key)
			if got != tt.expected {
				t.Errorf("isSecretKey(%q) = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}
}

func TestSplitScopeKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"scopes/default/db/password", "db/password"},
		{"scopes/staging/rabbitmq/password", "rabbitmq/password"},
		{"scopes/default/search/elasticsearch/auth/password", "search/elasticsearch/auth/password"},
		{"db/password", ""},      // no scope prefix
		{"scopes/", ""},          // incomplete
		{"scopes/default/", ""},  // scope but no key after
		{"scopes/default", ""},   // no trailing slash
		{"", ""},                 // empty
		{"other/prefix/key", ""}, // wrong prefix
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitScopeKey(tt.input)
			if got != tt.expected {
				t.Errorf("splitScopeKey(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CompareVersions — additional edge cases
// ---------------------------------------------------------------------------

func TestCompareVersions_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		v1, v2   string
		expected int
	}{
		{"non-numeric segment", "8.4-beta", "8.4", -1}, // atoi("4-beta") = 0, so compares as 8.0 vs 8.4
		{"both empty", "", "", 0},
		{"one empty", "1.0", "", 1},
		{"other empty", "", "1.0", -1},
		{"single segments equal", "8", "8", 0},
		{"single segments differ", "9", "8", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareVersions(tt.v1, tt.v2)
			if got != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UnresolvedTag — what makes a malformed template loud instead of silent
// ---------------------------------------------------------------------------
