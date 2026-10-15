// Package golden pins what the version matrix and the setup presets answer,
// for every platform, across the versions that matter. It was written before
// the matrix moved from Go into XML so that the move could be shown to change
// nothing it did not mean to: the file in testdata is the answer of the Go
// code as it was, and every difference since is a deliberate edit to it.
//
//	go test ./src/model/versions/golden/ -update
//
// rewrites the file from the current code. Review the diff before committing:
// a changed line is a project that will be set up differently.
package golden

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/faradey/madock/v4/src/helper/preset"
	"github.com/faradey/madock/v4/src/model/versions"
	_ "github.com/faradey/madock/v4/src/model/versions/bigcommerce"
	_ "github.com/faradey/madock/v4/src/model/versions/custom"
	_ "github.com/faradey/madock/v4/src/model/versions/magento2"
	_ "github.com/faradey/madock/v4/src/model/versions/medusa"
	_ "github.com/faradey/madock/v4/src/model/versions/prestashop"
	_ "github.com/faradey/madock/v4/src/model/versions/saleor"
	_ "github.com/faradey/madock/v4/src/model/versions/shopify"
	_ "github.com/faradey/madock/v4/src/model/versions/shopware"
	_ "github.com/faradey/madock/v4/src/model/versions/spree"
	_ "github.com/faradey/madock/v4/src/model/versions/sylius"
	_ "github.com/faradey/madock/v4/src/model/versions/woocommerce"
)

var update = flag.Bool("update", false, "rewrite testdata/matrix.golden.json from the current code")

// probes are the platform versions each platform is asked about: every
// boundary in the old Go code, one version either side of it, patch
// releases, and the ones a preset names.
var probes = map[string][]string{
	"magento2": {
		"", "1.9.0", "2.0.0", "2.1.0", "2.2.0", "2.2.9", "2.3.0", "2.3.1", "2.3.2", "2.3.3", "2.3.4", "2.3.5", "2.3.6",
		"2.3.7", "2.3.7-p3", "2.3.7-p4", "2.3.7-p5", "2.4.0", "2.4.1", "2.4.2", "2.4.3", "2.4.3-p1", "2.4.3-p2",
		"2.4.3-p3", "2.4.4", "2.4.5", "2.4.6", "2.4.6-p4", "2.4.7", "2.4.7-p4", "2.4.7-p5", "2.4.7-p10",
		"2.4.8", "2.4.8-p1", "2.4.9", "2.4.10", "2.5.0",
	},
	"shopware":    {"", "6.2", "6.3", "6.4", "6.4.20.2", "6.5", "6.5.8.0", "6.6", "6.7.13.1", "6.10.0.0"},
	"sylius":      {"", "1.9", "1.11", "1.12", "1.13", "1.14", "2.0", "2.1"},
	"custom":      {""},
	"prestashop":  {"", "8.1"},
	"woocommerce": {"", "9.0"},
	"medusa":      {"", "1", "2", "2.0"},
	"saleor":      {"", "3.20", "3.23"},
	"spree":       {"", "4.10", "5.0"},
	"shopify":     {"", "hydrogen", "app-remix", "api-php", "laravel-shopify"},
	"bigcommerce": {"", "catalyst", "stencil", "api-php", "app-node"},
}

type snapshot struct {
	Matrix  map[string]map[string]versions.ToolsVersions `json:"matrix"`
	Presets map[string][]presetEntry                     `json:"presets"`
}

type presetEntry struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Versions    versions.ToolsVersions `json:"versions"`
}

func current(t *testing.T) snapshot {
	t.Helper()
	// The magento2 provider reads composer.json from the working directory
	// when asked about no version; this directory has none.
	t.Setenv("MADOCK_RUN_DIR", t.TempDir())

	s := snapshot{Matrix: map[string]map[string]versions.ToolsVersions{}, Presets: map[string][]presetEntry{}}
	for platform, vers := range probes {
		s.Matrix[platform] = map[string]versions.ToolsVersions{}
		for _, v := range vers {
			got, ok := versions.GetVersionsForPlatform(platform, v)
			if !ok {
				t.Fatalf("no provider registered for %s", platform)
			}
			s.Matrix[platform][v] = got
		}
		for _, p := range preset.GetPresetsByPlatform(platform) {
			s.Presets[platform] = append(s.Presets[platform], presetEntry{p.Name, p.Description, p.Versions})
		}
	}
	return s
}

func TestMatrixAndPresetsMatchTheGolden(t *testing.T) {
	got, err := json.MarshalIndent(current(t), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join("testdata", "matrix.golden.json")

	if *update {
		if err := os.WriteFile(file, append(got, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}

	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("%v — run with -update to create it", err)
	}
	if string(want) != string(got)+"\n" {
		t.Errorf("the matrix or the presets answer differently from %s.\nRun `go test ./src/model/versions/golden/ -update` and review the diff: every changed line is a project set up differently.", file)
	}
}
