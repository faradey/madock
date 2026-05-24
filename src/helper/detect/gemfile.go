package detect

import (
	"os"
	"regexp"
	"strings"

	"github.com/faradey/madock/v3/src/helper/paths"
)

// DetectFromGemfile tries to detect Spree (and future Ruby platforms)
// from Gemfile or Gemfile.lock. Falls back to a substring scan instead
// of a full Bundler parser — Spree dependencies in a host Rails app
// look like `gem "spree"` or `gem 'spree', ...` and either form is
// enough to flag the platform.
func DetectFromGemfile(projectPath string) DetectionResult {
	result := DetectionResult{Detected: false}

	for _, name := range []string{"Gemfile.lock", "Gemfile"} {
		full := projectPath + "/" + name
		if !paths.IsFileExist(full) {
			continue
		}
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		content := string(data)
		if !rbHasSpree(content, name) {
			continue
		}
		result.Platform = "spree"
		result.Language = "ruby"
		result.PlatformVersion = rbExtractSpreeVersion(content, name)
		result.Detected = true
		result.Source = name
		return result
	}

	return result
}

func rbHasSpree(content, file string) bool {
	lower := strings.ToLower(content)
	if file == "Gemfile" {
		if matched, _ := regexp.MatchString(`(?mi)^\s*gem\s+["']spree["']`, content); matched {
			return true
		}
		// Spree starter Gemfile commonly groups under `gem "spree", ...`
		if strings.Contains(lower, `gem "spree"`) || strings.Contains(lower, `gem 'spree'`) {
			return true
		}
		return false
	}
	if file == "Gemfile.lock" {
		// In Gemfile.lock the resolved gem appears at the start of a
		// line as `    spree (5.0.0)`.
		if matched, _ := regexp.MatchString(`(?m)^\s{4}spree\s+\([0-9]`, content); matched {
			return true
		}
	}
	return false
}

func rbExtractSpreeVersion(content, file string) string {
	if file == "Gemfile.lock" {
		re := regexp.MustCompile(`(?m)^\s{4}spree\s+\(([0-9][^)]+)\)`)
		if m := re.FindStringSubmatch(content); len(m) == 2 {
			return cleanVersion(m[1])
		}
	}
	// Gemfile-side declarations rarely carry an exact version; resolver
	// owns it. Returning empty triggers the preset selector.
	return ""
}
