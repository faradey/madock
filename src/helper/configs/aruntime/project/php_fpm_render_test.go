package project

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// The golden fixtures pin the defaults, which is not the same as proving the
// setting works: a key nothing reads renders the default forever and looks
// right in every fixture.
//
// The default that matters is the cap. Until 4.0.3 the pool kept the
// distribution's `pm.max_children = 5`, which is plenty for a CLI and short of
// what an admin panel opens at once — measured on the Shopware 6.7 stand, where
// a cold administration load put several hundred assets and dozens of API calls
// in flight and everything past the fifth came back 503. It does not read as
// slow; it reads as a broken application, and half an hour went into telling it
// apart from a plugin defect.
func TestPhpFpmPoolIsConfigurable(t *testing.T) {
	dockerfile := func(t *testing.T, overrides map[string]string) string {
		t.Helper()

		projectName := "fpmpool"
		env := testenv.SetupWith(t, projectName, "fpmpool.test", overrides)

		MakeConf(projectName)

		rendered := testenv.Collect(t, filepath.Join(env.ExecDir, "aruntime", "projects", projectName), env)
		for name, body := range rendered {
			if strings.HasSuffix(name, "ctx/php.Dockerfile") {
				return body
			}
		}
		t.Fatal("MakeConf produced no php.Dockerfile")
		return ""
	}

	t.Run("the default raises the cap and leaves the rest alone", func(t *testing.T) {
		body := dockerfile(t, map[string]string{"php/enabled": "true"})

		if !strings.Contains(body, "pm.max_children = 40") {
			t.Errorf("the pool cap is not set, so it stays at the distribution's 5:\n%s", body)
		}

		// Deliberately the distribution's own numbers: these cost memory while
		// nothing is happening, and they were not what produced the 503s — with
		// pm = dynamic a burst spawns up to the cap as it arrives.
		for _, want := range []string{"pm.start_servers = 2", "pm.min_spare_servers = 1", "pm.max_spare_servers = 3"} {
			if !strings.Contains(body, want) {
				t.Errorf("%q is missing:\n%s", want, body)
			}
		}
	})

	t.Run("every value can be set", func(t *testing.T) {
		body := dockerfile(t, map[string]string{
			"php/enabled":               "true",
			"php/fpm/max_children":      "80",
			"php/fpm/start_servers":     "8",
			"php/fpm/min_spare_servers": "4",
			"php/fpm/max_spare_servers": "16",
		})

		for _, want := range []string{
			"pm.max_children = 80",
			"pm.start_servers = 8",
			"pm.min_spare_servers = 4",
			"pm.max_spare_servers = 16",
		} {
			if !strings.Contains(body, want) {
				t.Errorf("%q did not reach the image:\n%s", want, body)
			}
		}
	})
}
