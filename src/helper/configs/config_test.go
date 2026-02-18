package configs

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// evaluateCondition
// ---------------------------------------------------------------------------

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"true", "true", true},
		{"false", "false", false},
		{"TRUE uppercase", "TRUE", true},
		{"False mixed case", "False", false},
		{"true with spaces", "  true  ", true},
		{"false with spaces", "  false  ", false},
		{"truetrue repeated", "truetrue", true},
		{"truefalse mixed", "truefalse", false},
		{"unresolved placeholder", "{{{some/key}}}", false},
		{"arbitrary non-empty value", "somevalue", true},
		{"numeric value", "42", true},
		{"whitespace only", "   ", false},
		{"true with surrounding text", "enabled_true_flag", true},
		{"contains false substring", "not_false_at_all", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateCondition(tt.input)
			if got != tt.expected {
				t.Errorf("evaluateCondition(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// findMatchingEndif
// ---------------------------------------------------------------------------

func TestFindMatchingEndif(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		startPos int
		expected int
	}{
		{
			"simple no nesting",
			"content<<<endif>>>after",
			0,
			7, // "content" is 7 chars
		},
		{
			"nested one level",
			"<<<iftrue>>>inner<<<endif>>><<<endif>>>",
			0,
			28, // after the inner <<<endif>>> (12+5+11=28)
		},
		{
			"no endif returns -1",
			"content without closing tag",
			0,
			-1,
		},
		{
			"empty string returns -1",
			"",
			0,
			-1,
		},
		{
			"double nesting",
			// startPos is after outer <<<if>>>; content has one nested <<<if>>>...<<<endif>>>
			"<<<iffalse>>>deep<<<endif>>>mid<<<endif>>>end",
			0,
			31,
		},
		{
			"startPos skips content",
			"<<<endif>>><<<endif>>>",
			11, // start after first endif
			11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findMatchingEndif(tt.str, tt.startPos)
			if got != tt.expected {
				t.Errorf("findMatchingEndif(%q, %d) = %d, want %d", tt.str, tt.startPos, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// findElseAtDepth
// ---------------------------------------------------------------------------

func TestFindElseAtDepth_NoElse(t *testing.T) {
	got := findElseAtDepth("just some content")
	if got != -1 {
		t.Errorf("got %d, want -1", got)
	}
}

func TestFindElseAtDepth_TopLevel(t *testing.T) {
	content := "before<<<else>>>after"
	got := findElseAtDepth(content)
	if got != 6 {
		t.Errorf("got %d, want 6", got)
	}
}

func TestFindElseAtDepth_NestedElseIgnored(t *testing.T) {
	content := "<<<iftrue>>>inner<<<else>>>innerelse<<<endif>>><<<else>>>outerelse"
	got := findElseAtDepth(content)
	expected := len("<<<iftrue>>>inner<<<else>>>innerelse<<<endif>>>")
	if got != expected {
		t.Errorf("got %d, want %d", got, expected)
	}
}

func TestFindElseAtDepth_Empty(t *testing.T) {
	got := findElseAtDepth("")
	if got != -1 {
		t.Errorf("got %d, want -1", got)
	}
}

func TestFindElseAtDepth_DoubleNestedElseIgnored(t *testing.T) {
	// Two levels of nesting, each with an else — only the outermost should be found
	content := "<<<iftrue>>><<<iffalse>>>A<<<else>>>B<<<endif>>>C<<<else>>>OUTER<<<endif>>>D<<<else>>>TOP"
	got := findElseAtDepth(content)
	// Everything before the top-level <<<else>>>
	prefix := "<<<iftrue>>><<<iffalse>>>A<<<else>>>B<<<endif>>>C<<<else>>>OUTER<<<endif>>>D"
	if got != len(prefix) {
		t.Errorf("got %d, want %d", got, len(prefix))
	}
}

func TestFindElseAtDepth_NoTopLevelElse(t *testing.T) {
	// else exists only inside nested if — should return -1
	content := "<<<iftrue>>>X<<<else>>>Y<<<endif>>>Z"
	got := findElseAtDepth(content)
	if got != -1 {
		t.Errorf("got %d, want -1 (else is inside nested if)", got)
	}
}

// ---------------------------------------------------------------------------
// processConditionals — basic
// ---------------------------------------------------------------------------

func TestProcessConditionals_SimpleTrue(t *testing.T) {
	input := "before<<<iftrue>>>content<<<endif>>>after"
	expected := "beforecontentafter"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_SimpleFalse(t *testing.T) {
	input := "before<<<iffalse>>>content<<<endif>>>after"
	expected := "beforeafter"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_NoConditionals(t *testing.T) {
	input := "plain text with no conditionals"
	got := processConditionals(input)
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestProcessConditionals_EmptyString(t *testing.T) {
	got := processConditionals("")
	if got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}

func TestProcessConditionals_EmptyBody(t *testing.T) {
	input := "A<<<iftrue>>><<<endif>>>B"
	expected := "AB"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_MalformedUnclosedIf(t *testing.T) {
	input := "<<<iftrue>>>no closing tag"
	got := processConditionals(input)
	if got != input {
		t.Errorf("malformed input should be returned unchanged, got %q", got)
	}
}

func TestProcessConditionals_MalformedUnclosedTag(t *testing.T) {
	input := "<<<iftrueunclosed tag marker"
	got := processConditionals(input)
	if got != input {
		t.Errorf("malformed input should be returned unchanged, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// processConditionals — else branches
// ---------------------------------------------------------------------------

func TestProcessConditionals_ElseTrue(t *testing.T) {
	input := "<<<iftrue>>>yes<<<else>>>no<<<endif>>>"
	expected := "yes"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_ElseFalse(t *testing.T) {
	input := "<<<iffalse>>>yes<<<else>>>no<<<endif>>>"
	expected := "no"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_ElseWithSurroundingContent(t *testing.T) {
	input := "before<<<iftrue>>>A<<<else>>>B<<<endif>>>after"
	expected := "beforeAafter"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}

	input2 := "before<<<iffalse>>>A<<<else>>>B<<<endif>>>after"
	expected2 := "beforeBafter"
	got2 := processConditionals(input2)
	if got2 != expected2 {
		t.Errorf("got %q, want %q", got2, expected2)
	}
}

func TestProcessConditionals_EmptyTrueBranchWithElse(t *testing.T) {
	input := "<<<iftrue>>><<<else>>>B<<<endif>>>"
	expected := ""
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_EmptyFalseBranchWithElse(t *testing.T) {
	input := "<<<iffalse>>>A<<<else>>><<<endif>>>"
	expected := ""
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_NoElseNoChange(t *testing.T) {
	input := "<<<iftrue>>>kept<<<endif>>>"
	expected := "kept"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

// ---------------------------------------------------------------------------
// processConditionals — nesting
// ---------------------------------------------------------------------------

func TestProcessConditionals_NestedIfInsideElse(t *testing.T) {
	input := "<<<iffalse>>>A<<<else>>>B<<<iftrue>>>C<<<endif>>>D<<<endif>>>"
	expected := "BCD"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_NestedIfInsideTrueBranch(t *testing.T) {
	input := "<<<iftrue>>>A<<<iffalse>>>B<<<else>>>C<<<endif>>>D<<<endif>>>"
	expected := "ACD"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_NestedElseSkippedAtWrongDepth(t *testing.T) {
	input := "<<<iffalse>>>X<<<iftrue>>>Y<<<else>>>Z<<<endif>>>W<<<else>>>OUTER<<<endif>>>"
	expected := "OUTER"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_TripleNesting(t *testing.T) {
	// outer=true, mid=false (takes else), inner=true
	input := "<<<iftrue>>>L1<<<iffalse>>>L2a<<<else>>>L2b<<<iftrue>>>L3<<<endif>>>L2c<<<endif>>>L1end<<<endif>>>"
	expected := "L1L2bL3L2cL1end"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_SequentialBlocks(t *testing.T) {
	input := "<<<iftrue>>>A<<<endif>>>-<<<iffalse>>>B<<<else>>>C<<<endif>>>-<<<iftrue>>>D<<<else>>>E<<<endif>>>"
	expected := "A-C-D"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

// ---------------------------------------------------------------------------
// processConditionals — multiline / whitespace
// ---------------------------------------------------------------------------

func TestProcessConditionals_MultilineElse(t *testing.T) {
	input := `    environment:
<<<iftrue>>>
      USER: admin
      PASS: secret
<<<else>>>
      ANON: true
<<<endif>>>
    volumes:`
	expected := `    environment:

      USER: admin
      PASS: secret

    volumes:`
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_MultilineElseFalse(t *testing.T) {
	input := `    environment:
<<<iffalse>>>
      USER: admin
      PASS: secret
<<<else>>>
      ANON: true
<<<endif>>>
    volumes:`
	expected := `    environment:

      ANON: true

    volumes:`
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_FalseBlockRemovesLeadingWhitespace(t *testing.T) {
	input := "line1\n  <<<iffalse>>>removed<<<endif>>>\nline3"
	expected := "line1\nline3"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_InlineIfWithContentBefore(t *testing.T) {
	// Content before <<<if on same line — should only remove the block, not the line prefix
	input := "prefix <<<iffalse>>>removed<<<endif>>>suffix"
	expected := "prefix suffix"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

// ---------------------------------------------------------------------------
// processConditionals — grafana snippet simulation
// ---------------------------------------------------------------------------

func TestProcessConditionals_GrafanaSnippetAuthDisabled(t *testing.T) {
	input := `      GF_SERVE_FROM_SUB_PATH: true
      GF_SERVER_PROTOCOL: http
<<<iffalse>>>
      GF_SECURITY_ADMIN_USER: admin
      GF_SECURITY_ADMIN_PASSWORD: secret
<<<else>>>
      GF_AUTH_ANONYMOUS_ORG_ROLE: "Admin"
      GF_AUTH_ANONYMOUS_ENABLED: true
      GF_AUTH_BASIC_ENABLED: false
      GF_AUTH_DISABLE_LOGIN_FORM: true
<<<endif>>>
      GF_PATHS_PROVISIONING: /etc/grafana/provisioning`

	got := processConditionals(input)

	if strings.Contains(got, "GF_SECURITY_ADMIN_USER") {
		t.Error("auth disabled: should not contain GF_SECURITY_ADMIN_USER")
	}
	if !strings.Contains(got, "GF_AUTH_ANONYMOUS_ENABLED") {
		t.Error("auth disabled: should contain GF_AUTH_ANONYMOUS_ENABLED")
	}
	if !strings.Contains(got, "GF_SERVE_FROM_SUB_PATH") {
		t.Error("should preserve content outside conditional")
	}
	if !strings.Contains(got, "GF_PATHS_PROVISIONING") {
		t.Error("should preserve content after conditional")
	}
}

func TestProcessConditionals_GrafanaSnippetAuthEnabled(t *testing.T) {
	input := `      GF_SERVE_FROM_SUB_PATH: true
      GF_SERVER_PROTOCOL: http
<<<iftrue>>>
      GF_SECURITY_ADMIN_USER: admin
      GF_SECURITY_ADMIN_PASSWORD: secret
<<<else>>>
      GF_AUTH_ANONYMOUS_ORG_ROLE: "Admin"
      GF_AUTH_ANONYMOUS_ENABLED: true
      GF_AUTH_BASIC_ENABLED: false
      GF_AUTH_DISABLE_LOGIN_FORM: true
<<<endif>>>
      GF_PATHS_PROVISIONING: /etc/grafana/provisioning`

	got := processConditionals(input)

	if !strings.Contains(got, "GF_SECURITY_ADMIN_USER") {
		t.Error("auth enabled: should contain GF_SECURITY_ADMIN_USER")
	}
	if strings.Contains(got, "GF_AUTH_ANONYMOUS_ENABLED") {
		t.Error("auth enabled: should not contain GF_AUTH_ANONYMOUS_ENABLED")
	}
}

// ---------------------------------------------------------------------------
// isSecretKey / splitScopeKey
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
		{"db/password", ""},           // no scope prefix
		{"scopes/", ""},               // incomplete
		{"scopes/default/", ""},       // scope but no key after
		{"scopes/default", ""},        // no trailing slash
		{"", ""},                      // empty
		{"other/prefix/key", ""},      // wrong prefix
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
		{"non-numeric segment", "8.4-beta", "8.4", -1},       // atoi("4-beta") = 0, so compares as 8.0 vs 8.4
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
