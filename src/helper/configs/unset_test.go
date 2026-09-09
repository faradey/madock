package configs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/testenv"
)

// writeProjectFileConfig puts a committed `.madock/config.xml` beside the
// project's source, which is the layer that wins every key it declares.
func writeProjectFileConfig(t *testing.T, runDir string, data map[string]string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(runDir, ".madock"), 0o755); err != nil {
		t.Fatalf("creating .madock: %v", err)
	}
	configs.SaveInFile(filepath.Join(runDir, ".madock", "config.xml"), data, "default")
	configs.CleanCache()
}

// The case the feature was written for: a project ships two hostnames and one of
// them has no business on this machine.
//
// Before this, the committed file won every key it declared and the only ways to
// drop a host were editing a file that belongs to the repository or keeping a
// fork of it — measured on the demo server, which answered on the production
// hostnames because the repository says so.
func TestAMachineRemovesAHostTheRepositoryShips(t *testing.T) {
	env := testenv.SetupWith(t, "unsetproject", "", map[string]string{
		"unset/nginx/hosts/www": "1",
	})
	writeProjectFileConfig(t, env.RunDir, map[string]string{
		"nginx/hosts/base/name": "kept.test",
		"nginx/hosts/www/name":  "www.production.test",
		"nginx/hosts/www/code":  "www",
	})

	config := configs.GetProjectConfig("unsetproject")

	if value, still := config["nginx/hosts/www/name"]; still {
		t.Errorf("the host the machine removed is still there as %q", value)
	}
	// The whole host, not the key that was named: a host is several keys, and
	// leaving `code` behind would leave `GetHosts` describing half a host.
	if value, still := config["nginx/hosts/www/code"]; still {
		t.Errorf("only part of the host was removed — code is still %q", value)
	}
	if config["nginx/hosts/base/name"] != "kept.test" {
		t.Errorf("the neighbouring host went with it: %q", config["nginx/hosts/base/name"])
	}
}

// A path names a subtree. Anything else makes the common case a list of keys
// somebody has to keep complete.
//
// The assertion is on the values rather than on the keys' absence, and that is
// the point of the layer semantics: `php/ini/*` has shipped defaults, so
// removing the repository's numbers does not leave a hole — it lets the
// defaults through. A key disappears only when no lower layer defines it, which
// is the host case above.
func TestUnsetTakesTheWholeSubtree(t *testing.T) {
	env := testenv.SetupWith(t, "unsetsubtree", "", map[string]string{
		"unset/php/ini": "1",
	})
	writeProjectFileConfig(t, env.RunDir, map[string]string{
		"php/ini/post_max_size":       "64M",
		"php/ini/upload_max_filesize": "64M",
		"php/ini/max_input_vars":      "5000",
		"public_dir":                  "pub",
	})

	config := configs.GetProjectConfig("unsetsubtree")

	for key, repositoryValue := range map[string]string{
		"php/ini/post_max_size":       "64M",
		"php/ini/upload_max_filesize": "64M",
		"php/ini/max_input_vars":      "5000",
	} {
		if config[key] == repositoryValue {
			t.Errorf("%s is still the repository's %q — the subtree was not removed", key, repositoryValue)
		}
		if config[key] == "" {
			t.Errorf("%s is empty: the shipped default should have shown through", key)
		}
	}
	if config["public_dir"] != "pub" {
		t.Errorf("a key outside the subtree was removed too: %q", config["public_dir"])
	}
}

// The half that makes `add` unnecessary: what an unset removes is the layer
// **above** the file declaring it, so the machine's own value — which had been
// losing to the repository since layering existed — is what applies afterwards.
//
// Written as its own test because the alternative reading, "delete the key from
// the finished configuration", passes every other test in this file and fails
// here: it would leave nothing behind and the template would render an empty
// string.
func TestTheMachinesOwnValueAppliesAfterTheUnset(t *testing.T) {
	env := testenv.SetupWith(t, "unsetmachinewins", "", map[string]string{
		// testenv writes 128M into the machine's own file.
		"unset/proxy/max_body_size": "1",
	})
	writeProjectFileConfig(t, env.RunDir, map[string]string{
		"proxy/max_body_size": "512M",
	})

	config := configs.GetProjectConfig("unsetmachinewins")

	if config["proxy/max_body_size"] != "128M" {
		t.Errorf("max_body_size is %q, expected the machine's own 128M — "+
			"the repository's 512M should have been removed and nothing else", config["proxy/max_body_size"])
	}
}

// The declaration in the committed file is refused, and this is the half of the
// design that is about safety rather than layering: that file arrives by a
// `git pull` and a deploy, so a line in it would remove settings on every
// machine at once with nobody on those machines deciding anything.
//
// It also cannot work — an unset removes keys from the layers above the file
// that declares it, and nothing sits above the project's own file.
func TestUnsetInTheProjectFileIsIgnored(t *testing.T) {
	env := testenv.SetupWith(t, "unsetproject2", "", map[string]string{
		"proxy/max_body_size": "128M",
	})
	writeProjectFileConfig(t, env.RunDir, map[string]string{
		"unset/proxy/max_body_size": "1",
		"nginx/hosts/base/name":     "kept.test",
	})

	config := configs.GetProjectConfig("unsetproject2")

	if config["proxy/max_body_size"] != "128M" {
		t.Errorf("a committed <unset> removed a machine's setting: max_body_size is %q", config["proxy/max_body_size"])
	}
	if config["nginx/hosts/base/name"] != "kept.test" {
		t.Errorf("the rest of the committed file stopped being read: %q", config["nginx/hosts/base/name"])
	}
}

// Removing one of these does not produce an error, it produces a stack whose
// templates interpolate an empty string — a vhost with no root, an image with no
// version. The line is refused and named instead.
func TestProtectedKeysSurviveUnset(t *testing.T) {
	env := testenv.SetupWith(t, "unsetprotected", "", map[string]string{
		"unset/php/version": "1",
		"unset/platform":    "1",
	})
	writeProjectFileConfig(t, env.RunDir, map[string]string{
		"php/version": "8.3",
		"platform":    "magento2",
	})

	config := configs.GetProjectConfig("unsetprotected")

	if config["php/version"] != "8.3" {
		t.Errorf("php/version was unset: %q", config["php/version"])
	}
	if config["platform"] != "magento2" {
		t.Errorf("platform was unset: %q", config["platform"])
	}
}

// The declarations are instructions, not settings. Left in the map they reach
// every template that iterates the configuration, and a key named
// `unset/nginx/hosts/www` rendered into a compose file is the kind of thing that
// is found on a server rather than here.
func TestDeclarationsNeverReachTheConfiguration(t *testing.T) {
	env := testenv.SetupWith(t, "unsetclean", "", map[string]string{
		"unset/php/ini/max_input_vars": "1",
	})
	writeProjectFileConfig(t, env.RunDir, map[string]string{
		"php/ini/max_input_vars": "5000",
	})

	config := configs.GetProjectConfig("unsetclean")

	for key := range config {
		if strings.HasPrefix(key, "unset/") {
			t.Errorf("a declaration reached the configuration: %s", key)
		}
	}
	if config["php/ini/max_input_vars"] == "5000" {
		t.Error("and it did not do its job either: the repository's max_input_vars is still in force")
	}
}

// `config:unset --machine` and a hand-edited file have to produce the same
// thing, or the command becomes a second way of writing the feature that drifts
// from the first.
func TestDeclareUnsetWritesWhatTheReaderLooksFor(t *testing.T) {
	env := testenv.SetupWith(t, "unsetdeclare", "", nil)
	writeProjectFileConfig(t, env.RunDir, map[string]string{
		"nginx/hosts/www/name": "www.production.test",
	})

	if err := configs.DeclareUnset("unsetdeclare", "nginx/hosts/www", "default"); err != nil {
		t.Fatalf("declaring: %v", err)
	}
	configs.CleanCache()

	config := configs.GetProjectConfig("unsetdeclare")
	if value, still := config["nginx/hosts/www/name"]; still {
		t.Errorf("the declaration was written and did nothing: host is still %q", value)
	}

	declared := configs.DeclaredUnsets("unsetdeclare")
	if len(declared) != 1 || declared[0] != "nginx/hosts/www" {
		t.Errorf("config:list would report %v, expected [nginx/hosts/www]", declared)
	}
}

// The refusal is in the command as well as in the reader, so a devops learns it
// at the moment of typing rather than from a value that quietly stayed.
func TestDeclareUnsetRefusesAProtectedKey(t *testing.T) {
	testenv.SetupWith(t, "unsetdeclareprotected", "", nil)

	err := configs.DeclareUnset("unsetdeclareprotected", "php/version", "default")
	if err == nil {
		t.Fatal("php/version was accepted for unset")
	}
	if !strings.Contains(err.Error(), "php/version") {
		t.Errorf("the refusal does not name the key: %v", err)
	}
}
