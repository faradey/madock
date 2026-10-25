//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAnImportedMagentoGetsItsOwnVersionAndItsSearchEngine — an existing
// 2.4.8-p5 shop set up with `--platform magento2`, the way the Indemorio
// engine imports one (found 2026-10-25):
//
//   - naming the platform skipped reading the version from the code, so the
//     shop got the newest release's PHP (8.5), which its lock refuses;
//   - setup:env wrote Elasticsearch into env.php whatever the project ran, so
//     on OpenSearch — every 2.4.8+ — setup:upgrade found "No alive nodes".
func TestAnImportedMagentoGetsItsOwnVersionAndItsSearchEngine(t *testing.T) {
	p := newProject(t, "e2eimport")
	lock := `{"packages": [{"name": "magento/product-community-edition", "version": "2.4.8-p5"}]}`
	if err := os.WriteFile(filepath.Join(p.runDir, "composer.lock"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(p.runDir, "app", "etc"), 0o755); err != nil {
		t.Fatal(err)
	}

	out := p.run(setupTimeout, "setup", "-y", "--platform=magento2", "--hosts=e2eimport.test")
	requireContains(t, out, "Detected magento2 2.4.8-p5", "the version read from the code although the platform was named")
	conf := p.run(time.Minute, "config:list")
	requireContains(t, conf, "php/version 8.4", "the PHP the matrix gives 2.4.8, not the newest release's")
	requireContains(t, conf, "search/opensearch/enabled true", "OpenSearch, which 2.4.8 runs")

	p.run(30*time.Minute, "start")
	// No shop in the database yet: setup:env writes nothing, and has to say
	// so — the script used die(), which exits 0, and madock reported success.
	if out, err := p.tryRun(5*time.Minute, "setup:env", "--force"); err == nil {
		t.Errorf("setup:env with no core_config_data reported success:\n%s", out)
	}

	// The few tables of an imported shop that setup:env reads.
	p.freshTable("store", "(store_id INT PRIMARY KEY, code VARCHAR(32), website_id INT)")
	p.freshTable("store_website", "(website_id INT PRIMARY KEY, code VARCHAR(32), is_default INT)")
	p.freshTable("core_config_data", "(config_id INT AUTO_INCREMENT PRIMARY KEY, scope VARCHAR(8), scope_id INT, path VARCHAR(255), value TEXT)")
	p.query("INSERT INTO store VALUES (0, 'admin', 0), (1, 'default', 1)")
	p.query("INSERT INTO store_website VALUES (0, 'admin', 0), (1, 'base', 1)")
	p.query("INSERT INTO core_config_data (scope, scope_id, path, value) VALUES ('default', 0, 'web/unsecure/base_url', 'https://shop.example.com/')")
	p.run(5*time.Minute, "setup:env", "--force")
	env, err := os.ReadFile(filepath.Join(p.runDir, "app", "etc", "env.php"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"'engine' => 'opensearch'", "'opensearch_server_hostname' => 'opensearch'", "'opensearch_server_port' => '9200'"} {
		if !strings.Contains(string(env), want) {
			t.Errorf("env.php has no %s:\n%s", want, firstLines(string(env), 80))
		}
	}
	if strings.Contains(string(env), "elasticsearch7_server_hostname") {
		t.Error("env.php still names elasticsearch for a project on OpenSearch")
	}
}
