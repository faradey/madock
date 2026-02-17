package project

import (
	"os"
	"testing"

	"github.com/faradey/madock/v3/src/helper/configs"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		// Equal versions
		{
			name:     "equal simple versions",
			v1:       "1.0.0",
			v2:       "1.0.0",
			expected: 0,
		},
		{
			name:     "equal two-part versions",
			v1:       "10.4",
			v2:       "10.4",
			expected: 0,
		},

		// v1 > v2
		{
			name:     "major version greater",
			v1:       "2.0.0",
			v2:       "1.0.0",
			expected: 1,
		},
		{
			name:     "minor version greater",
			v1:       "1.5.0",
			v2:       "1.4.0",
			expected: 1,
		},
		{
			name:     "patch version greater",
			v1:       "1.0.5",
			v2:       "1.0.4",
			expected: 1,
		},
		{
			name:     "MariaDB version comparison",
			v1:       "10.6",
			v2:       "10.4",
			expected: 1,
		},
		{
			name:     "double digit version greater",
			v1:       "11.4",
			v2:       "10.6",
			expected: 1,
		},

		// v1 < v2
		{
			name:     "major version less",
			v1:       "1.0.0",
			v2:       "2.0.0",
			expected: -1,
		},
		{
			name:     "minor version less",
			v1:       "1.4.0",
			v2:       "1.5.0",
			expected: -1,
		},
		{
			name:     "patch version less",
			v1:       "1.0.4",
			v2:       "1.0.5",
			expected: -1,
		},
		{
			name:     "MariaDB version less",
			v1:       "10.4",
			v2:       "10.6",
			expected: -1,
		},

		// Different length versions
		{
			name:     "shorter version equals longer with zeros",
			v1:       "1.0",
			v2:       "1.0.0",
			expected: 0,
		},
		{
			name:     "shorter version less than longer",
			v1:       "1.0",
			v2:       "1.0.1",
			expected: -1,
		},
		{
			name:     "longer version greater",
			v1:       "1.0.1",
			v2:       "1.0",
			expected: 1,
		},

		// Edge cases
		{
			name:     "empty versions",
			v1:       "",
			v2:       "",
			expected: 0,
		},
		{
			name:     "single number versions",
			v1:       "2",
			v2:       "1",
			expected: 1,
		},
		{
			name:     "PHP version comparison",
			v1:       "8.2",
			v2:       "8.1",
			expected: 1,
		},
		{
			name:     "PHP 8 vs 7",
			v1:       "8.1",
			v2:       "7.4",
			expected: 1,
		},
		{
			name:     "Elasticsearch versions",
			v1:       "8.4.3",
			v2:       "7.17.5",
			expected: 1,
		},
		{
			name:     "Redis versions",
			v1:       "7.2.3",
			v2:       "6.2",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := configs.CompareVersions(tt.v1, tt.v2)
			if got != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.expected)
			}
		})
	}
}

func TestCompareVersionsSymmetry(t *testing.T) {
	// Test that comparing v1 vs v2 gives opposite result of v2 vs v1
	versions := []string{"1.0.0", "2.0.0", "10.4", "10.6", "8.1", "8.2"}

	for i := 0; i < len(versions); i++ {
		for j := 0; j < len(versions); j++ {
			v1, v2 := versions[i], versions[j]
			r1 := configs.CompareVersions(v1, v2)
			r2 := configs.CompareVersions(v2, v1)

			if r1 != -r2 {
				t.Errorf("Symmetry broken: CompareVersions(%q, %q)=%d but CompareVersions(%q, %q)=%d",
					v1, v2, r1, v2, v1, r2)
			}
		}
	}
}

func TestProcessSnippets(t *testing.T) {
	// Test with no snippets
	input := []byte("This is a test without snippets")
	result := ProcessSnippets(input, "testproject")
	if string(result) != string(input) {
		t.Errorf("ProcessSnippets should return unchanged input when no snippets present")
	}

	// Test with empty input
	result = ProcessSnippets([]byte(""), "testproject")
	if string(result) != "" {
		t.Errorf("ProcessSnippets should return empty string for empty input")
	}
}

func TestProcessSnippetsPattern(t *testing.T) {
	// Test that the regex pattern matches correctly
	tests := []struct {
		input    string
		hasMatch bool
	}{
		{"{{{include snippets/test.txt}}}", true},
		{"{{{include snippets/path/to/file.conf}}}", true},
		{"no snippet here", false},
		{"{{{other/pattern}}}", false},
		{"{{{include other/path}}}", false},
	}

	for _, tt := range tests {
		// We can't fully test ProcessSnippets without file system setup,
		// but we can verify it doesn't panic on various inputs
		func() {
			defer func() {
				if r := recover(); r != nil && !tt.hasMatch {
					// Expected to not panic for non-matching inputs
				}
			}()

			if !tt.hasMatch {
				result := ProcessSnippets([]byte(tt.input), "testproject")
				if string(result) != tt.input {
					t.Errorf("Non-matching input should be unchanged: %s", tt.input)
				}
			}
		}()
	}
}

func TestGetDockerConfigFilePathPriority(t *testing.T) {
	// This test documents the path priority order that GetDockerConfigFile uses:
	// 1. {runDir}/.madock/docker/{path}
	// 2. {execDir}/projects/{projectName}/docker/{path}
	// 3. {execDir}/docker/{platform}/{path}
	// 4. {execDir}/docker/general/service/{path}

	// We can't fully test without mocking paths, but we document the expected behavior
	t.Log("GetDockerConfigFile checks paths in order:")
	t.Log("1. Project .madock override: {runDir}/.madock/docker/{path}")
	t.Log("2. Project-specific: {execDir}/projects/{projectName}/docker/{path}")
	t.Log("3. Platform-specific: {execDir}/docker/{platform}/{path}")
	t.Log("4. General service: {execDir}/docker/general/service/{path}")
}

func TestGetSnippetFilePathPriority(t *testing.T) {
	// This test documents the path priority order that GetSnippetFile uses:
	// 1. {runDir}/.madock/docker/{path}
	// 2. {execDir}/projects/{projectName}/docker/{path}
	// 3. {execDir}/docker/{path}

	t.Log("GetSnippetFile checks paths in order:")
	t.Log("1. Project .madock override: {runDir}/.madock/docker/{path}")
	t.Log("2. Project-specific: {execDir}/projects/{projectName}/docker/{path}")
	t.Log("3. Global docker: {execDir}/docker/{path}")
}

// Integration test for ProcessSnippets with actual file
func TestProcessSnippetsWithFile(t *testing.T) {
	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "processsnippets")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// This test is limited because ProcessSnippets depends on paths.GetRunDirPath()
	// and paths.GetExecDirPath() which we can't easily mock.
	// A more comprehensive test would require dependency injection.

	t.Log("ProcessSnippets requires file system access to fully test")
	t.Log("Consider refactoring to accept path resolver interface for better testability")
}
