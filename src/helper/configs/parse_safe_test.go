package configs

import (
	"os"
	"path/filepath"
	"testing"
)

// ParseXmlFileSafe is what lets the shared proxy.conf builder read every
// project's config without one unparsable file taking the whole command down:
// it returns the parse error instead of calling Fatalln. ParseXmlFile keeps the
// Fatalln for the single-project callers that should still stop.
func TestParseXmlFileSafeReturnsErrorInsteadOfExiting(t *testing.T) {
	dir := t.TempDir()

	good := filepath.Join(dir, "good.xml")
	if err := os.WriteFile(good, []byte("<config><marker>yes</marker></config>"), 0o644); err != nil {
		t.Fatal(err)
	}
	conf, err := ParseXmlFileSafe(good)
	if err != nil {
		t.Fatalf("a parsable file returned an error: %v", err)
	}
	if conf["marker"] != "yes" {
		t.Errorf("marker = %q, want yes", conf["marker"])
	}

	bad := filepath.Join(dir, "bad.xml")
	if err := os.WriteFile(bad, []byte("<config><scopes><default>not closed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseXmlFileSafe(bad); err == nil {
		t.Error("an unparsable file returned no error — the proxy builder would still Fatalln on it")
	}
}

// ListProjectsIn is read by the shared proxy.conf builder on every start, so a
// project whose config.xml does not parse has to be reported, not fatal: the old
// ParseXmlFile would os.Exit here and take the whole render — and this test
// process — down, which is the regression.
func TestListProjectsInReportsABrokenConfigInsteadOfExiting(t *testing.T) {
	execDir := t.TempDir()
	mk := func(name, body string) {
		dir := filepath.Join(execDir, "projects", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.xml"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("good", "<config><scopes><default><path>"+execDir+"</path></default></scopes></config>")
	mk("bad", "<config><scopes><default>not closed")

	byName := map[string]string{}
	for _, e := range ListProjectsIn(execDir) {
		byName[e.Name] = e.State
	}
	if byName["bad"] != ProjectBrokenConfig {
		t.Errorf("bad project state = %q, want %q", byName["bad"], ProjectBrokenConfig)
	}
	if _, ok := byName["good"]; !ok {
		t.Error("the good project was dropped along with the broken one")
	}
}
