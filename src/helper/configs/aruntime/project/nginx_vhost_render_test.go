package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// A project adds a location of its own by dropping one file into
// .madock/docker/nginx/vhost.d/. The golden fixtures render every platform
// with that directory absent, which is the common case and the one that proves
// nothing about the feature: an include of an empty directory looks the same
// whether or not anything renders into it.
func TestNginxVhostSnippetsAreRenderedAndMounted(t *testing.T) {
	projectName := "vhostd"
	env := testenv.SetupWith(t, projectName, "vhostd.test", nil)

	snippetDir := filepath.Join(env.RunDir, ".madock", "docker", "nginx", "vhost.d")
	if err := os.MkdirAll(snippetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Goes through the template engine like the vhost itself, so a snippet can
	// name the same values — {{{.workdir}}} here.
	snippet := "location = /healthz {\n    alias {{{.workdir}}}/pub/health.txt;\n}\n"
	if err := os.WriteFile(filepath.Join(snippetDir, "10-health.conf"), []byte(snippet), 0o644); err != nil {
		t.Fatal(err)
	}
	// Not a .conf: ignored, the way nginx's own include glob would ignore it.
	if err := os.WriteFile(filepath.Join(snippetDir, "README.md"), []byte("notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	MakeConf(projectName)

	ctx := filepath.Join(env.ExecDir, "aruntime", "projects", projectName, "ctx")

	body, err := os.ReadFile(filepath.Join(ctx, "nginx-vhost.d", "10-health.conf"))
	if err != nil {
		t.Fatalf("the snippet was not rendered into the build context: %v", err)
	}
	if strings.Contains(string(body), "{{{") {
		t.Errorf("the snippet was copied, not rendered:\n%s", body)
	}
	if !strings.Contains(string(body), "/pub/health.txt") {
		t.Errorf("the snippet lost its content on the way:\n%s", body)
	}
	if _, err := os.Stat(filepath.Join(ctx, "nginx-vhost.d", "README.md")); err == nil {
		t.Error("a file that is not a .conf was rendered into the mounted directory")
	}

	// The vhost has to include the directory inside its server block, and the
	// compose file has to mount it — a snippet rendered and mounted nowhere is
	// the same as no snippet.
	vhost, err := os.ReadFile(filepath.Join(ctx, "nginx.conf"))
	if err != nil {
		t.Fatal(err)
	}
	server := strings.Index(string(vhost), "server {")
	include := strings.Index(string(vhost), "include /etc/nginx/vhost.d/*.conf;")
	if server < 0 || include < server {
		t.Errorf("the vhost does not include vhost.d inside its server block:\n%s", vhost)
	}

	compose, err := os.ReadFile(filepath.Join(env.ExecDir, "aruntime", "projects", projectName, "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(compose), "./ctx/nginx-vhost.d:/etc/nginx/vhost.d") {
		t.Errorf("the compose file does not mount the rendered snippets:\n%s", compose)
	}
}

// Removing the file from the project has to remove the rule from nginx. The
// rendered copy lives in a directory the project never looks at, so a stale one
// would keep serving a location nobody can find in the repository.
func TestNginxVhostSnippetRemovedFromTheProjectLeavesTheContext(t *testing.T) {
	projectName := "vhostd"
	env := testenv.SetupWith(t, projectName, "vhostd.test", nil)

	snippetDir := filepath.Join(env.RunDir, ".madock", "docker", "nginx", "vhost.d")
	if err := os.MkdirAll(snippetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	snippetFile := filepath.Join(snippetDir, "20-old.conf")
	if err := os.WriteFile(snippetFile, []byte("location = /old { return 410; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	MakeConf(projectName)

	rendered := filepath.Join(env.ExecDir, "aruntime", "projects", projectName, "ctx", "nginx-vhost.d", "20-old.conf")
	if _, err := os.Stat(rendered); err != nil {
		t.Fatalf("nothing below is being tested — the snippet did not render in the first place: %v", err)
	}

	if err := os.Remove(snippetFile); err != nil {
		t.Fatal(err)
	}
	MakeConf(projectName)

	if _, err := os.Stat(rendered); err == nil {
		t.Error("the snippet is gone from the project and its rendered copy is still mounted into nginx")
	}
}

// With no snippets the directory still has to exist: the mount is
// unconditional, and docker creates a missing bind source itself, as root.
func TestNginxVhostDirectoryExistsWithoutSnippets(t *testing.T) {
	projectName := "vhostd"
	env := testenv.SetupWith(t, projectName, "vhostd.test", nil)

	MakeConf(projectName)

	info, err := os.Stat(filepath.Join(env.ExecDir, "aruntime", "projects", projectName, "ctx", "nginx-vhost.d"))
	if err != nil {
		t.Fatalf("the mounted directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("nginx-vhost.d is not a directory")
	}
}
