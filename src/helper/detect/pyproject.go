package detect

import (
	"os"
	"regexp"
	"strings"

	"github.com/faradey/madock/v3/src/helper/paths"
)

// DetectFromPyproject tries to detect Saleor (and future Python platforms)
// from pyproject.toml or requirements*.txt. Falls back to a substring scan
// instead of full TOML parsing — dependencies in Saleor's pyproject look
// like `saleor = "..."` or `"saleor"` and either form is enough to flag
// the platform.
func DetectFromPyproject(projectPath string) DetectionResult {
	result := DetectionResult{Detected: false}

	candidates := []string{"pyproject.toml", "requirements.txt", "uv.lock", "poetry.lock"}
	for _, name := range candidates {
		full := projectPath + "/" + name
		if !paths.IsFileExist(full) {
			continue
		}
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		content := string(data)
		if !pyHasSaleor(content, name) {
			continue
		}
		result.Platform = "saleor"
		result.Language = "python"
		result.PlatformVersion = pyExtractSaleorVersion(content)
		result.Detected = true
		result.Source = name
		return result
	}

	return result
}

func pyHasSaleor(content, file string) bool {
	lower := strings.ToLower(content)
	if file == "pyproject.toml" {
		// Look for `name = "saleor"` or a `saleor` dependency line.
		if strings.Contains(lower, "name = \"saleor\"") {
			return true
		}
		if strings.Contains(lower, "\"saleor\"") {
			return true
		}
	}
	if file == "requirements.txt" {
		if matched, _ := regexp.MatchString(`(?mi)^saleor([=<>~!\s].*)?$`, content); matched {
			return true
		}
	}
	if file == "uv.lock" || file == "poetry.lock" {
		if strings.Contains(lower, "name = \"saleor\"") {
			return true
		}
	}
	return false
}

func pyExtractSaleorVersion(content string) string {
	// Best-effort: read `version = "3.23.6"` from the [project] / [tool.poetry]
	// table that owns the `name = "saleor"` field. Returns empty when the
	// version can't be located — the setup wizard will then fall back to
	// the preset selector.
	re := regexp.MustCompile(`(?ms)name\s*=\s*"saleor".*?version\s*=\s*"([^"]+)"`)
	if m := re.FindStringSubmatch(content); len(m) == 2 {
		return cleanVersion(m[1])
	}
	return ""
}
