package install

import (
	"os/exec"
	"strings"
	"testing"
)

func magentoConf(engine string) map[string]string {
	conf := map[string]string{
		"db/database":              "magento",
		"db/user":                  "magento",
		"db/password":              "pw",
		"magento/admin_first_name": "A",
		"magento/admin_last_name":  "B",
		"magento/admin_email":      "a@example.com",
		"magento/admin_user":       "admin",
		"magento/admin_password":   "admin123",
		"magento/admin_frontname":  "admin",
		"magento/locale":           "en_US",
		"magento/currency":         "USD",
		"magento/timezone":         "UTC",
	}
	if engine != "" {
		conf["search/"+engine+"/enabled"] = "true"
	}
	return conf
}

const installHead = "bin/magento setup:install --base-url=https://shop.test --db-host=db --db-name=magento --db-user=magento --db-password=pw --admin-firstname=A --admin-lastname=B --admin-email=a@example.com --admin-user=admin --admin-password=admin123 --backend-frontname=admin --language=en_US --currency=USD --timezone=UTC --use-rewrites=1 "

const installTail = " && bin/magento setup:upgrade && bin/magento cache:clean && bin/magento indexer:reindex | bin/magento cache:flush"

// Without a provider the install line is, character for character, the one
// madock ran before the seam: written out here from that code.
func TestMagentoInstallLineIsUnchangedWithoutAProvider(t *testing.T) {
	saved := searchInstallArgsProviders
	searchInstallArgsProviders = nil
	t.Cleanup(func() { searchInstallArgsProviders = saved })

	cases := []struct {
		engine, version, want string
	}{
		{"opensearch", "2.4.8", installHead +
			"--search-engine=opensearch --opensearch-host=opensearch --opensearch-port=9200 --opensearch-index-prefix=magento2 --opensearch-timeout=15 " +
			"&& bin/magento module:disable Magento_AdminAdobeImsTwoFactorAuth && bin/magento module:disable Magento_TwoFactorAuth " + installTail},
		{"elasticsearch", "2.4.8", installHead +
			"--search-engine=elasticsearch7 --elasticsearch-host=elasticsearch --elasticsearch-port=9200 --elasticsearch-index-prefix=magento2 --elasticsearch-timeout=15 " +
			"&& bin/magento module:disable Magento_AdminAdobeImsTwoFactorAuth && bin/magento module:disable Magento_TwoFactorAuth " + installTail},
		{"opensearch", "2.4.5", installHead +
			"--search-engine=elasticsearch7 --elasticsearch-host=opensearch --elasticsearch-port=9200 --elasticsearch-index-prefix=magento2 --elasticsearch-timeout=15 " +
			"&& bin/magento module:disable Magento_TwoFactorAuth " + installTail},
		{"", "2.4.8", installHead +
			"&& bin/magento module:disable Magento_AdminAdobeImsTwoFactorAuth && bin/magento module:disable Magento_TwoFactorAuth " + installTail},
	}
	for _, c := range cases {
		if got := magentoInstallCommand(magentoConf(c.engine), "shop.test", c.version); got != c.want {
			t.Errorf("%s on %s:\ngot  %q\nwant %q", c.engine, c.version, got, c.want)
		}
	}
}

// A provider's arguments follow the engine's flags, it learns which flag
// family and which service the line uses, and a value of any character reaches
// Magento as typed through the shell.
func TestSearchInstallArgsFollowTheEngineFlagsQuoted(t *testing.T) {
	saved := searchInstallArgsProviders
	searchInstallArgsProviders = nil
	t.Cleanup(func() { searchInstallArgsProviders = saved })

	const password = `p'w$x "y" \z`
	var seen []string
	RegisterSearchInstallArgs(func(_ map[string]string, family, service string) []string {
		seen = append(seen, family+"/"+service)
		return []string{"--" + family + "-enable-auth=1", "--" + family + "-password=" + password}
	})

	cases := []struct{ engine, version, family, service string }{
		{"opensearch", "2.4.8", "opensearch", "opensearch"},
		{"elasticsearch", "2.4.8", "elasticsearch", "elasticsearch"},
		{"opensearch", "2.4.5", "elasticsearch", "opensearch"},
	}
	for _, c := range cases {
		seen = nil
		line := magentoInstallCommand(magentoConf(c.engine), "shop.test", c.version)
		if len(seen) != 1 || seen[0] != c.family+"/"+c.service {
			t.Errorf("%s %s: provider saw %v, want %s/%s", c.engine, c.version, seen, c.family, c.service)
		}
		timeout := "--" + c.family + "-timeout=15 "
		at := strings.Index(line, timeout)
		if at < 0 || !strings.HasPrefix(line[at+len(timeout):], "'--"+c.family+"-enable-auth=1' ") {
			t.Errorf("%s %s: the arguments do not follow the engine flags:\n%s", c.engine, c.version, line)
		}

		// What the shell hands the program: the setup:install part only.
		cmd := line[:strings.Index(line, " &&")]
		out, err := exec.Command("bash", "-c", "printf '%s\\n' "+strings.TrimPrefix(cmd, "bin/magento setup:install ")).Output()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), "\n--"+c.family+"-password="+password+"\n") {
			t.Errorf("%s %s: the password did not reach the command as typed:\n%s", c.engine, c.version, out)
		}
	}

	// No search engine: nothing to log in to, and the provider is not asked.
	seen = nil
	magentoInstallCommand(magentoConf(""), "shop.test", "2.4.8")
	if len(seen) != 0 {
		t.Errorf("asked with no search engine: %v", seen)
	}
}
