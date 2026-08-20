package dockerassets

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/tmpl"
)

// TestEveryKeyIsAKeyMadockHas is what replaces missingkey=error.
//
// The obvious win of a real template engine was supposed to be that a mistyped
// key fails the render instead of quietly reading as false. It cannot: a shared
// snippet asks about memcached/enabled on platforms whose configuration has
// never heard of memcached, and the old engine answered false by leaving the
// placeholder standing. Making that fatal would break `madock start` on every
// such project, so an absent key stays falsy — and a typo would be just as
// silent as it was.
//
// This is the same check moved to where it belongs. It reads every key every
// template mentions and holds it against the keys madock actually has, for all
// eleven platforms at once, in a second, without starting anything. A render
// could only ever have checked the one platform somebody happened to run.
func TestEveryKeyIsAKeyMadockHas(t *testing.T) {
	known := knownKeys(t)

	unknown := map[string][]string{}
	err := fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		body, readErr := fs.ReadFile(FS, path)
		if readErr != nil {
			return readErr
		}

		keys, parseErr := tmpl.Keys(path, string(body))
		if parseErr != nil {
			return nil // TestEveryTemplateParses reports this one
		}

		for _, key := range keys {
			if !known[key] {
				unknown[key] = append(unknown[key], path)
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded templates: %v", err)
	}

	for _, key := range sorted(unknown) {
		t.Errorf("no such setting: %q, read by %s", key, strings.Join(unknown[key], ", "))
	}
}

// knownKeys is every setting madock can hold.
//
// Two sources, because madock has two. config.xml carries the defaults every
// project starts from, and the per-platform settings are written by the setup
// that asks for them — shopware/messenger/enabled exists nowhere but in the Go
// line that sets it. Reading the Go source for those literals is coarse and it
// is honest about being coarse: this test proves a key exists somewhere, not
// that the platform being rendered has it.
func knownKeys(t *testing.T) map[string]bool {
	t.Helper()
	root := repoRoot(t)
	known := map[string]bool{}

	// config_defaults.xml is the one compiled into the binary and is the
	// authority; config.xml at the root is the copy an installation gets and
	// can be edited, so it is read too rather than assumed to agree.
	for _, file := range []string{
		filepath.Join(root, "src", "helper", "configs", "config_defaults.xml"),
		filepath.Join(root, "config.xml"),
	} {
		config, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		for _, key := range xmlPaths(string(config)) {
			known[key] = true
		}
	}

	keyLiteral := regexp.MustCompile(`"([a-z][a-z0-9_]*(?:/[a-z0-9_]+)+)"`)
	err := filepath.WalkDir(filepath.Join(root, "src"), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return walkErr
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range keyLiteral.FindAllStringSubmatch(string(body), -1) {
			known[match[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking src: %v", err)
	}

	// What no configuration file holds and no Go literal spells as a path: the
	// values the renderer computes for every template. They are listed rather
	// than discovered, so that adding one is a decision somebody wrote down.
	for _, key := range []string{
		"main_service",          // which container the front door points at
		"main_service_enabled",  // and whether it is part of the stack at all
		"main_service_port",     // the port to proxy to when it is not php
		"main_upstream_server",  // host:port of the shared proxy's upstream
		"project_name",          // lowercased, as containers are named
		"scope",                 // the active scope, as a name suffix
		"nginx/hosts",           // the ordered host list, never empty
		// A block whose children are named by whoever writes them, like
		// nginx/hosts above. Declaring the parent in config.xml is not the
		// answer and was tried: an empty <programs></programs> makes it a
		// value, and the first worker/programs/<name> then fails to build a
		// tree under a leaf.
		"worker/programs",
		"nginx/http2/directive", // computed in the proxy generator, not a setting
		"proxy/rate_limit/req",  // likewise
		"os/arch", "os/user/uid", "os/user/name", "os/user/guid", "os/user/ugroup",
	} {
		known[key] = true
	}

	return known
}

// xmlPaths reads config.xml as a list of slash-separated leaf paths. It is a
// tag scanner and not a parser on purpose: what a leaf holds does not matter
// here, only that the key exists.
func xmlPaths(body string) []string {
	tag := regexp.MustCompile(`<(/?)([a-z][a-z0-9_]*)>`)
	var stack []string
	var paths []string

	for _, match := range tag.FindAllStringSubmatch(withoutComments(body), -1) {
		if match[1] == "/" {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		}
		stack = append(stack, match[2])
		// Everything under <scopes><default> is a setting; the two wrappers and
		// the <config> root are not part of the key.
		if len(stack) > 3 && stack[0] == "config" && stack[1] == "scopes" {
			paths = append(paths, strings.Join(stack[3:], "/"))
		}
	}

	return paths
}

// withoutComments removes XML comments before the tags are counted.
//
// The scanner below is a regexp over tag names and has no idea what a comment
// is, so a tag mentioned inside one is pushed onto the stack and — having no
// closing half — never leaves it. Every key after that point is computed one
// level deep and the test reports that madock has no setting called
// `restart_policy`, which is forty lines of nonsense pointing nowhere near the
// comment that caused it.
//
// It was already latent: config_defaults.xml carries commented-out `<hosts>`
// and `<jobs>` blocks, which happen to be balanced and so cancel out. A comment
// that merely names a tag in prose does not.
func withoutComments(body string) string {
	comment := regexp.MustCompile(`(?s)<!--.*?-->`)
	return comment.ReplaceAllString(body, "")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate this test file")
	}
	return filepath.Dir(filepath.Dir(file))
}

func sorted(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// TestXmlPathsIgnoresComments pins the above. Without it the failure lands on
// every key in the file except the one that caused it.
func TestXmlPathsIgnoresComments(t *testing.T) {
	keys := xmlPaths(`<?xml version="1.0"?>
<config>
    <!-- Outside <scopes> on purpose, and this sentence used to break the scan.
         So did a commented-out block:
         <hosts>
             <host>example.test</host> -->
    <allow_destructive_commands>true</allow_destructive_commands>
    <scopes>
        <default>
            <restart_policy>no</restart_policy>
            <php>
                <version>8.2</version>
            </php>
        </default>
    </scopes>
</config>`)

	found := map[string]bool{}
	for _, key := range keys {
		found[key] = true
	}

	for _, want := range []string{"restart_policy", "php/version"} {
		if !found[want] {
			t.Errorf("%q was not read; the comment shifted the scan: %v", want, keys)
		}
	}
	if found["allow_destructive_commands"] {
		t.Error("a top-level key was read as a setting; it lives outside <scopes> and is not one")
	}
}
