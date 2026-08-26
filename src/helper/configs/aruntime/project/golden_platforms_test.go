package project

import (
	"path/filepath"
	"testing"

	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/configs/projects"
	"github.com/faradey/madock/v4/src/helper/testenv"
	"github.com/faradey/madock/v4/src/model/versions"

	// The version providers register themselves in init(), and in the real
	// binary they arrive with the platform controllers. The controllers cannot
	// be imported here — they import this package — so the leaf packages are
	// named directly. Without them GetVersionsForPlatform answers "no such
	// platform", `language` goes unwritten, and every fixture quietly renders
	// through the general fallback: that is exactly how Medusa's application
	// image came to be pinned as php-fpm.
	_ "github.com/faradey/madock/v4/src/model/versions/bigcommerce"
	_ "github.com/faradey/madock/v4/src/model/versions/medusa"
	_ "github.com/faradey/madock/v4/src/model/versions/prestashop"
	_ "github.com/faradey/madock/v4/src/model/versions/saleor"
	_ "github.com/faradey/madock/v4/src/model/versions/shopify"
	_ "github.com/faradey/madock/v4/src/model/versions/shopware"
	_ "github.com/faradey/madock/v4/src/model/versions/spree"
	_ "github.com/faradey/madock/v4/src/model/versions/sylius"
)

// platformsWithoutAGolden is every platform madock ships templates for and had
// no rendered fixture: magento2, custom and woocommerce are covered by the
// cases in golden_test.go.
//
// The gap they left is narrow and worth stating exactly. `TestEveryTemplateParses`
// proves each platform's templates are syntactically valid and the key audit
// proves they reference keys that exist — neither renders one. A template that
// parses can still produce a compose file with a service pointing at nothing, an
// entrypoint for a framework that does not have it, or an image tag that never
// existed; that is what a rendered fixture shows and a parse cannot.
var platformsWithoutAGolden = []string{
	"bigcommerce",
	"medusa",
	"prestashop",
	"saleor",
	"shopify",
	"shopware",
	"spree",
	"sylius",
}

// goldenToolVersions is one fixed set of versions for every platform fixture.
//
// Fixed rather than read from the shipped defaults on purpose: those move with
// upstream releases, and a golden that changes when a default changes reports a
// version bump as a rendering difference — which is how a fixture stops being
// read. What each platform does with these is the thing under test; whether
// PHP 8.4 is still current is a question `freshness` answers elsewhere.
var goldenToolVersions = versions.ToolsVersions{
	Php:          "8.4",
	Composer:     "2",
	Db:           "11.4",
	SearchEngine: "OpenSearch",
	Elastic:      "8.11.14",
	OpenSearch:   "2.12.0",
	Redis:        "7.2",
	Valkey:       "8.1",
	RabbitMQ:     "3.13",
	Xdebug:       "3.4.4",
	NodeJs:       "22.14.0",
	Yarn:         "1.22.22",
	Python:       "3.12",
	Golang:       "1.24",
	Ruby:         "3.3",
}

// TestGoldenPlatformStacks renders one project per platform and pins the result.
//
// The configuration is written by the platform's own env writer rather than by a
// list of keys here, and that is the difference between this and a fixture that
// merely says `platform: saleor`. Saleor turns PHP off and Python on, moves the
// upstream port to 8000 and insists on PostgreSQL; Shopify picks a preset and
// half its presets have no PHP at all. Restating any of that here would pin what
// the test author believed instead of what `setup` does, and the two drift
// silently — the fixture would keep passing while the real thing changed.
func TestGoldenPlatformStacks(t *testing.T) {
	for _, platform := range platformsWithoutAGolden {
		t.Run(platform, func(t *testing.T) {
			projectName := "golden"
			env := testenv.SetupWith(t, projectName, "golden.test", map[string]string{"platform": platform})

			writePlatformDefaults(t, env.ExecDir, projectName, platform)

			MakeConf(projectName)

			rendered := testenv.Collect(t, filepath.Join(env.ExecDir, "aruntime", "projects", projectName), env)
			if len(rendered) == 0 {
				t.Fatal("MakeConf produced no files")
			}

			goldenDir := filepath.Join("testdata", "golden", "platform-"+platform)
			if *updateGolden {
				testenv.WriteGolden(t, goldenDir, rendered)
				return
			}
			testenv.CompareGolden(t, goldenDir, rendered)
		})
	}
}

// writePlatformDefaults runs the platform's own config writer over the project,
// the way `setup` does once the questions have been answered.
func writePlatformDefaults(t *testing.T, execDir, projectName, platform string) {
	t.Helper()

	writer, ok := projects.GetEnvWriter(platform)
	if !ok {
		t.Fatalf("no config writer is registered for platform %q", platform)
	}

	config := new(configs.ConfigLines)
	config.EnvFile = filepath.Join(execDir, "projects", projectName, "config.xml")
	config.ActiveScope = "default"

	// `platform` and `language` are written by the shared path that setup goes
	// through before it reaches the platform's own writer, and this fixture used
	// to skip them — which made every golden less representative than it looked.
	//
	// `language` decides which template tree answers when a platform ships no
	// file of its own: the resolver tries the platform, then
	// docker/languages/<language>/, then docker/general/service/. With the key
	// empty, the language step is skipped and everything lands on the general
	// one — so Medusa, a Node platform with no Dockerfile of its own, had its
	// application image pinned here as **php-fpm on ubuntu**, and the fixture
	// would have gone on approving that if the real resolution ever broke.
	config.Set("platform", platform)

	defaults, ok := versions.GetVersionsForPlatform(platform, "")
	if !ok {
		t.Fatalf("no version provider is registered for %q, so this fixture would render through the "+
			"general fallback and pin whatever that produces — add the blank import above", platform)
	}
	if defaults.Language != "" {
		config.Set("language", defaults.Language)
	}

	// The base configuration stands in for the general one, which is what the
	// writers read defaults out of — credentials, timezone, the switches a
	// platform does not decide for itself.
	base := configs.GetProjectConfigOnly(projectName)
	writer(config, goldenToolVersions, base, map[string]string{})
	config.Save()

	// The rendering reads the configuration back through the cache, and the file
	// has just changed underneath it.
	configs.CleanCache()
}
