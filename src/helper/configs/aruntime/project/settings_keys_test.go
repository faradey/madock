package project

import (
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// The settings below were literals inside templates until 4.3.3. The golden
// fixtures render their defaults, which are those literals, so what these
// prove is the other half: a project that sets one gets its value.

func TestVarnishSizeIsASetting(t *testing.T) {
	env := testenv.SetupWith(t, "varnishsize", "varnishsize.test", map[string]string{
		"varnish/enabled": "true",
		"varnish/size":    "256M",
	})

	MakeConf("varnishsize")

	compose := readGenerated(t, env, "docker-compose.yml")
	if !strings.Contains(compose, "VARNISH_SIZE: 256M") {
		t.Errorf("varnish/size did not reach the compose file:\n%s", compose)
	}
}

// grunt-cli was installed into every image that has node, whether or not the
// project builds anything with grunt. Emptying the default has to remove it
// everywhere, and extra has to arrive everywhere.
func TestNpmGlobalPackagesAreSettings(t *testing.T) {
	env := testenv.SetupWith(t, "npmglobal", "npmglobal.test", map[string]string{
		"nodejs/enabled":          "true",
		"nodejs/embedded/enabled": "true",
		"nodejs/npm/default":      "",
		"nodejs/npm/extra":        "pnpm",
	})

	MakeConf("npmglobal")

	for file, want := range map[string]string{
		"ctx/php.Dockerfile":    "RUN { npm install -g pnpm",
		"ctx/nodejs.Dockerfile": "RUN { npm install -g pnpm",
		"ctx/claude.Dockerfile": "RUN { npm install -g pnpm",
	} {
		body := readGenerated(t, env, file)
		if !strings.Contains(body, want) {
			t.Errorf("%s: missing %q:\n%s", file, want, body)
		}
		if strings.Contains(body, "grunt-cli") {
			t.Errorf("%s: still installs grunt-cli with nodejs/npm/default empty:\n%s", file, body)
		}
	}

	// Node as the main service is a different template from the node service
	// beside Magento: snippets/dockerfile/nodejs/packages, not nodejs/nodejs.
	env = testenv.SetupWith(t, "npmglobalnode", "npmglobalnode.test", map[string]string{
		"platform":           "custom",
		"language":           "nodejs",
		"php/enabled":        "false",
		"nodejs/enabled":     "true",
		"nodejs/npm/default": "",
		"nodejs/npm/extra":   "pnpm",
	})

	MakeConf("npmglobalnode")

	body := readGenerated(t, env, "ctx/nodejs.Dockerfile")
	if !strings.Contains(body, "RUN { npm install -g pnpm") || strings.Contains(body, "grunt-cli") {
		t.Errorf("the nodejs language image ignores nodejs/npm/*:\n%s", body)
	}
}

// The dashboards were pinned to linux/x86_64 on every machine, so on an arm64
// Mac they ran emulated even where an arm64 image exists. Only versions
// published for amd64 alone ask for it now. Checked on Docker Hub 2026-10-08:
// kibana has arm64 from 7.16, opensearch-dashboards from 1.1.
func TestDashboardsRunNativeWhereAnImageExists(t *testing.T) {
	for _, tc := range []struct {
		engine, version, service string
		amd64                    bool
	}{
		{"elasticsearch", "7.10.1", "kibana", true},
		{"elasticsearch", "7.17.5", "kibana", false},
		{"opensearch", "1.0.0", "opensearchdashboard", true},
		{"opensearch", "2.19.1", "opensearchdashboard", false},
	} {
		t.Run(tc.service+"-"+tc.version, func(t *testing.T) {
			name := "dash" + strings.ReplaceAll(tc.engine+tc.version, ".", "")
			env := testenv.SetupWith(t, name, name+".test", map[string]string{
				"search/engine":                             tc.engine,
				"search/" + tc.engine + "/enabled":           "true",
				"search/" + tc.engine + "/version":           tc.version,
				"search/" + tc.engine + "/dashboard/enabled": "true",
			})

			MakeConf(name)

			compose := readGenerated(t, env, "docker-compose.yml")
			start := strings.Index(compose, "  "+tc.service+":")
			if start < 0 {
				t.Fatalf("no %s service in the compose file:\n%s", tc.service, compose)
			}
			block := compose[start:]
			if end := strings.Index(block, "restart:"); end > 0 {
				block = block[:end]
			}
			if strings.Contains(block, "x86_64") {
				t.Errorf("%s %s still pinned to x86_64:\n%s", tc.service, tc.version, block)
			}
			if got := strings.Contains(block, "platform: linux/amd64"); got != tc.amd64 {
				t.Errorf("%s %s: platform linux/amd64 rendered = %v, want %v:\n%s", tc.service, tc.version, got, tc.amd64, block)
			}
		})
	}
}

func TestMessengerLimitsAreSettings(t *testing.T) {
	for _, tc := range []struct{ platform, want string }{
		{"shopware", "--time-limit=600 --memory-limit=1G"},
		{"sylius", "--memory-limit=1G --time-limit=600"},
	} {
		t.Run(tc.platform, func(t *testing.T) {
			name := "messenger" + tc.platform
			env := testenv.SetupWith(t, name, name+".test", map[string]string{
				"platform":                              tc.platform,
				"language":                              "php",
				tc.platform + "/messenger/enabled":      "true",
				tc.platform + "/messenger/memory_limit": "1G",
				tc.platform + "/messenger/time_limit":   "600",
			})

			MakeConf(name)

			compose := readGenerated(t, env, "docker-compose.yml")
			if !strings.Contains(compose, tc.want) {
				t.Errorf("the messenger limits did not reach the compose file, want %q:\n%s", tc.want, compose)
			}
		})
	}
}
