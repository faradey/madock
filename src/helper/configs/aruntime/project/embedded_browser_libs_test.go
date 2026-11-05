package project

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// The snippet that installs a browser's libraries is in every image that embeds
// Node, and it read php/browser_libs, so a Go project with PHP disabled had to set
// a PHP key to get them. nodejs/embedded/browser_libs is the key now, for any
// language, and the old name keeps working for the projects that set it.
func TestEmbeddedNodeBrowserLibs(t *testing.T) {
	installsThem := func(t *testing.T, overrides map[string]string) bool {
		t.Helper()

		projectName := "browserlibs"
		env := testenv.SetupWith(t, projectName, "browserlibs.test", overrides)

		MakeConf(projectName)

		rendered := testenv.Collect(t, filepath.Join(env.ExecDir, "aruntime", "projects", projectName), env)
		for _, body := range rendered {
			if strings.Contains(body, "RUN npx --yes playwright install-deps chromium") {
				return true
			}
		}
		return false
	}

	language := func(name string, extra map[string]string) map[string]string {
		overrides := map[string]string{
			"platform":                "custom",
			"language":                name,
			"php/enabled":             "false",
			"nodejs/embedded/enabled": "true",
		}
		for key, value := range extra {
			overrides[key] = value
		}
		return overrides
	}

	on := map[string]string{"nodejs/embedded/browser_libs": "true"}
	cases := []struct {
		name      string
		overrides map[string]string
		want      bool
	}{
		{"off by default in a Go image", language("golang", nil), false},
		{"the new key in a Go image", language("golang", on), true},
		{"the new key in a Python image", language("python", on), true},
		{"the new key in a Ruby image", language("ruby", on), true},
		{"the new key in a PHP image", map[string]string{"nodejs/embedded/enabled": "true", "nodejs/embedded/browser_libs": "true"}, true},
		{"the old name in a PHP image", map[string]string{"nodejs/embedded/enabled": "true", "php/browser_libs": "true"}, true},
		{"the old name in a Go image", language("golang", map[string]string{"php/browser_libs": "true"}), true},
		{"the standalone container's key does not reach this image", language("golang", map[string]string{"nodejs/browser_libs": "true"}), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := installsThem(t, c.overrides); got != c.want {
				t.Errorf("playwright install-deps rendered: %v, want %v", got, c.want)
			}
		})
	}
}
