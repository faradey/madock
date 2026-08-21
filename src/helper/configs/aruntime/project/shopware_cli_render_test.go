package project

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// The golden fixtures render Shopware with shopware-cli off, which is the
// default and the state most projects stay in. That leaves the half that
// actually does something untested: a feature gated behind a config key is
// exactly the kind that renders nothing forever and is noticed a release later.
//
// shopware-cli is the vendor's extension tooling, and it is not part of a
// project the way bin/console is — composer cannot install a compiled Go binary,
// so the image has to carry it. Same shape as n98-magerun in the magento2 image.
func TestShopwareCliRendersOnlyWhenAskedFor(t *testing.T) {
	dockerfile := func(t *testing.T, overrides map[string]string) string {
		t.Helper()

		projectName := "swcli"
		env := testenv.SetupWith(t, projectName, "swcli.test", overrides)

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

	t.Run("off by default", func(t *testing.T) {
		body := dockerfile(t, map[string]string{"platform": "shopware"})

		if strings.Contains(body, "shopware-cli") {
			t.Errorf("shopware-cli was installed without being asked for:\n%s", body)
		}
	})

	t.Run("installed when enabled", func(t *testing.T) {
		body := dockerfile(t, map[string]string{
			"platform":             "shopware",
			"shopware/cli/enabled": "true",
			"shopware/cli/version": "0.17.3",
		})

		if !strings.Contains(body, "/usr/local/bin/shopware-cli") {
			t.Errorf("shopware/cli/enabled did not put the binary in the image:\n%s", body)
		}

		// The version reaches the URL. A gate that renders the block but drops
		// the version would fetch a 404 and unpack nothing, and the image would
		// build right up until the tar.
		if !strings.Contains(body, "releases/download/0.17.3/") {
			t.Errorf("the configured version is missing from the download URL:\n%s", body)
		}

		// Both architectures, because the vendor's names do not match uname -m:
		// x86_64 is "Linux_x86_64" but aarch64 is "Linux_arm64". Getting that
		// wrong fails only on the machine nobody built on.
		for _, want := range []string{"Linux_x86_64", "Linux_arm64"} {
			if !strings.Contains(body, want) {
				t.Errorf("no mapping for %s in the rendered Dockerfile:\n%s", want, body)
			}
		}
	})
}
