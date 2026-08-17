package project

import (
	"flag"
	"path/filepath"
	"testing"

	"github.com/faradey/madock/v3/src/helper/testenv"
)

// Run these with -count=1, always:
//
//	go test -count=1 ./src/helper/configs/aruntime/project/...
//
// Go caches a package's test result until one of its .go files changes. These
// tests read their input from docker/ — plain data files the cache knows
// nothing about — so editing a template and re-running reports `ok (cached)`
// and proves nothing. Measured: a deliberately reversed dev/start preference
// passed a cached run and failed immediately with -count=1. The pre-push hook
// passes the flag for this reason.
//
// updateGolden rewrites the expected files instead of comparing against them:
//
//	go test -count=1 ./src/helper/configs/aruntime/project/... -run Golden -update
//
// Review the diff it produces like any other change. A golden file that was
// updated without being read is worse than no test — it records whatever the
// code does today and calls it correct.
var updateGolden = flag.Bool("update", false, "rewrite golden files")

// Golden tests render a project's whole docker configuration and compare it,
// file by file, against a committed copy.
//
// They exist because most of what breaks in madock breaks here. Of the eight
// defects found in the week these were written, four were generation: a Node
// entrypoint that preferred `dev` where a server needs `start`, a Dockerfile
// that was never written for a Node service on a non-PHP project, a config
// change that did not reach the generated files until a rebuild, and an
// OpenSearch default pointing at a tag that does not exist. All four are the
// kind of thing a diff of the rendered output shows immediately.
//
// The point of a golden file over an assertion is that it fails on changes
// nobody thought to assert. An assertion answers the question you asked; a
// golden file answers the one you did not.
type goldenCase struct {
	name      string
	overrides map[string]string
}

func goldenCases() []goldenCase {
	return []goldenCase{
		{
			// The default shape: Magento on PHP with MariaDB and OpenSearch.
			name: "magento2-php",
		},
		{
			// Node inside the application container. Until 3.9.8 this was
			// php/nodejs/enabled and no golden case turned it on, so the whole
			// half of the php image that installs node was rendered by nothing
			// and compared against nothing.
			name: "magento2-php-embedded-node",
			overrides: map[string]string{
				"nodejs/embedded":     "true",
				"nodejs/yarn/enabled": "true",
			},
		},
		{
			// The reason the key was renamed. A Python service with a
			// JavaScript front end needs node in its own image exactly as a PHP
			// one does, and before 3.9.8 there was no way to ask for it.
			name: "custom-python-embedded-node",
			overrides: map[string]string{
				"platform":        "custom",
				"language":        "python",
				"php/enabled":     "false",
				"nodejs/embedded": "true",
			},
		},
		{
			// Two databases. The credentials of the second one are its own —
			// reading them from db/* is a defect this pins.
			name: "magento2-db2",
			overrides: map[string]string{
				"db2/enabled":       "true",
				"db2/root_password": "second_root",
				"db2/user":          "second_user",
				"db2/password":      "second_pw",
				"db2/database":      "second",
			},
		},
		{
			// PostgreSQL instead of MySQL: a different db service and a
			// different client in every command that talks to it.
			name: "magento2-postgres",
			overrides: map[string]string{
				"db/type":       "postgresql",
				"db/repository": "postgres",
				"db/version":    "16",
			},
		},
		{
			// PostgreSQL 18, which mounts its volume one level up from every
			// version before it. The image keeps a major-version directory
			// under /var/lib/postgresql, and finding data in the old path makes
			// it refuse to start — so the fixture exists to hold the mount, not
			// the version number.
			name: "magento2-postgres18",
			overrides: map[string]string{
				"db/type":       "postgresql",
				"db/repository": "postgres",
				"db/version":    "18",
			},
		},
		{
			// MongoDB, which is the engine that has to be told a cache size.
			// Left alone, WiredTiger sizes its cache from the RAM it can see —
			// the host's, not the container's limit — so one mongod on a large
			// machine reserves gigabytes nobody asked it to. This fixture is
			// here to hold the --wiredTigerCacheSizeGB the budget produces.
			name: "magento2-mongodb",
			overrides: map[string]string{
				"db/type":       "mongodb",
				"db/repository": "mongo",
				"db/version":    "7",
			},
		},
		{
			// Mail turned off. sendmail_path used to be written whatever the
			// configuration said, so with mailpit disabled every mail() call
			// went to a port nobody was listening on and failed silently. The
			// generated Dockerfile must simply not touch it.
			name: "magento2-no-sendmail",
			overrides: map[string]string{
				"php/sendmail/enabled": "false",
			},
		},
		{
			// With an envelope sender configured. Without one msmtp refuses a
			// plain mail() call, and the argument has to be absent rather than
			// empty when the address is not set — which is why it is rendered
			// whole rather than as a value.
			name: "magento2-sendmail-from",
			overrides: map[string]string{
				"php/sendmail/from": "shop@example.com",
			},
		},
		{
			// The sandbox shape — no language at all, so the main service is
			// "app" and there is no PHP container.
			name: "custom-none",
			overrides: map[string]string{
				"platform":    "custom",
				"language":    "none",
				"php/enabled": "false",
				"app/enabled": "true",
			},
		},
		{
			// Node as the main service. Its Dockerfile comes from the language
			// template, which carries cron where the service template does not.
			name: "custom-nodejs",
			overrides: map[string]string{
				"platform":       "custom",
				"language":       "nodejs",
				"php/enabled":    "false",
				"nodejs/enabled": "true",
			},
		},
		{
			// A Node service beside a language that is not Node. The compose
			// file renders the service on nodejs/enabled alone, and its
			// Dockerfile used to be written only for PHP projects — so this
			// case existed as a compose service pointing at a missing file.
			name: "custom-none-with-nodejs",
			overrides: map[string]string{
				"platform":       "custom",
				"language":       "none",
				"php/enabled":    "false",
				"app/enabled":    "true",
				"nodejs/enabled": "true",
			},
		},
		{
			// A php platform of its own. Pins the fastcgi HTTPS parameter, which
			// used to be the constant `on`: the application then believed every
			// request was secure, including one that arrived over plain http at the
			// published project port, and built https URLs and secure cookies for
			// it. Shopware and Sylius hardcoded SERVER_PORT 443 beside it.
			name: "woocommerce-php",
			overrides: map[string]string{
				"platform": "woocommerce",
				"language": "php",
			},
		},
		{
			// PHP switched on beside a language that is not PHP. The nginx
			// templates used to include a snippet per enabled runtime, so this
			// shape rendered two server blocks on the same listen and
			// server_name: nginx kept the php one and warned about the other,
			// and the route to the app answered 404 from a document root that
			// has no index.php. The front door follows the language, so the
			// only server block here proxies to app.
			name: "custom-none-with-php",
			overrides: map[string]string{
				"platform":    "custom",
				"language":    "none",
				"php/enabled": "true",
				"app/enabled": "true",
			},
		},
		{
			// A project that answers no request: the owner of a shared database
			// schema, a queue worker. No nginx container, and nothing that
			// depends on one — the compose file is the whole point of the case,
			// because a service left with `depends_on: nginx` is a file docker
			// refuses to read.
			name: "custom-none-no-nginx",
			overrides: map[string]string{
				"platform":        "custom",
				"language":        "none",
				"php/enabled":     "false",
				"app/enabled":     "true",
				"nginx/enabled":   "false",
				"varnish/enabled": "true",
			},
		},
		{
			// What a Node project on a server looks like: production, and a
			// named script rather than whatever package.json happens to have.
			name: "custom-nodejs-production",
			overrides: map[string]string{
				"platform":            "custom",
				"language":            "nodejs",
				"php/enabled":         "false",
				"nodejs/enabled":      "true",
				"nodejs/env":          "production",
				"nodejs/script":       "docker-start",
				"nodejs/script_type":  "command",
				"nodejs/browser_libs": "true",
			},
		},
	}
}

func TestGoldenGeneratedConfig(t *testing.T) {
	for _, testCase := range goldenCases() {
		t.Run(testCase.name, func(t *testing.T) {
			projectName := "golden"
			env := testenv.SetupWith(t, projectName, "golden.test", testCase.overrides)

			MakeConf(projectName)

			rendered := testenv.Collect(t, filepath.Join(env.ExecDir, "aruntime", "projects", projectName), env)
			if len(rendered) == 0 {
				t.Fatal("MakeConf produced no files")
			}

			goldenDir := filepath.Join("testdata", "golden", testCase.name)
			if *updateGolden {
				testenv.WriteGolden(t, goldenDir, rendered)
				return
			}
			testenv.CompareGolden(t, goldenDir, rendered)
		})
	}
}
