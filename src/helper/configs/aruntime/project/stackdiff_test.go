package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// A resolved compose config the way `docker compose config --format json`
// writes one: long-form volumes with absolute sources, build as an object.
// Two services build from the shared ctx, one is an image, and the source tree
// is mounted the way every platform mounts it.
func testStack(t *testing.T, runtimeDir string) []byte {
	t.Helper()
	ctx := filepath.Join(runtimeDir, "ctx")
	stack := map[string]any{
		"name": "madock_golden",
		"services": map[string]any{
			"nginx": map[string]any{
				"build": map[string]any{"context": ctx, "dockerfile": "nginx.Dockerfile"},
				"volumes": []map[string]any{
					{"type": "bind", "source": filepath.Join(runtimeDir, "src"), "target": "/var/www/html"},
					{"type": "bind", "source": filepath.Join(ctx, "nginx.conf"), "target": "/etc/nginx/conf.d/default.conf"},
					{"type": "bind", "source": filepath.Join(ctx, "nginx-vhost.d"), "target": "/etc/nginx/vhost.d"},
				},
			},
			"php": map[string]any{
				"build": map[string]any{"context": ctx, "dockerfile": "php.Dockerfile"},
				"volumes": []map[string]any{
					{"type": "bind", "source": filepath.Join(runtimeDir, "src"), "target": "/var/www/html"},
				},
			},
			"db": map[string]any{
				"image": "mariadb:11.4",
				"volumes": []map[string]any{
					{"type": "volume", "source": "dbdata", "target": "/var/lib/mysql"},
					{"type": "bind", "source": filepath.Join(ctx, "my.cnf"), "target": "/etc/mysql/my.cnf"},
				},
			},
		},
		"networks": map[string]any{"default": map[string]any{"name": "madock_golden_default"}},
		"volumes":  map[string]any{"dbdata": map[string]any{"name": "madock_golden_dbdata"}},
	}
	data, err := json.Marshal(stack)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeCtx(t *testing.T, runtimeDir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(runtimeDir, "ctx", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func baseCtx() map[string]string {
	return map[string]string{
		"nginx.Dockerfile":       "FROM nginx:1.27\n",
		"php.Dockerfile":         "FROM php:8.4-fpm\nRUN apt-get install -y curl\n",
		"nginx.conf":             "server { listen 80; }\n",
		"my.cnf":                 "[mysqld]\ninnodb_buffer_pool_size = 512M\n",
		"nginx-vhost.d/.gitkeep": "",
	}
}

func record(t *testing.T, runtimeDir string, ctx map[string]string) stackRecord {
	t.Helper()
	writeCtx(t, runtimeDir, ctx)
	if err := os.MkdirAll(filepath.Join(runtimeDir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	rec, err := recordFromComposeConfig(testStack(t, runtimeDir), runtimeDir)
	if err != nil {
		t.Fatalf("recording the stack: %v", err)
	}
	return rec
}

// The case the mechanism was built for: the vhost changes, nothing else does,
// and the answer has to be "nginx, files only" — not the database.
func TestVhostEditIsAnNginxRestartAndNothingElse(t *testing.T) {
	dir := t.TempDir()
	before := record(t, dir, baseCtx())

	ctx := baseCtx()
	ctx["nginx.conf"] = "server { listen 80; location = /healthz { return 200; } }\n"
	after := record(t, dir, ctx)

	diff := diffStacks(before, after)
	want := StackDiff{Known: true, Restart: []string{"nginx"}}
	if !reflect.DeepEqual(diff, want) {
		t.Errorf("a vhost edit produced %+v, want %+v", diff, want)
	}
	if !diff.Narrow() {
		t.Error("a vhost edit is not narrowed to nginx")
	}
}

// A snippet dropped into the mounted directory is the same kind of change: the
// directory is bind-mounted whole, so a new file in it is a mount change.
func TestVhostSnippetIsAnNginxRestart(t *testing.T) {
	dir := t.TempDir()
	before := record(t, dir, baseCtx())

	ctx := baseCtx()
	ctx["nginx-vhost.d/10-health.conf"] = "location = /healthz { return 200; }\n"
	after := record(t, dir, ctx)

	diff := diffStacks(before, after)
	if !reflect.DeepEqual(diff.Restart, []string{"nginx"}) || len(diff.Recreate) != 0 || diff.Global {
		t.Errorf("a new vhost.d snippet produced %+v", diff)
	}
}

// The other half of a mounted file: a database setting is a db restart, and
// still not a recreate of anything.
func TestMyCnfEditIsADbRestart(t *testing.T) {
	dir := t.TempDir()
	before := record(t, dir, baseCtx())

	ctx := baseCtx()
	ctx["my.cnf"] = "[mysqld]\ninnodb_buffer_pool_size = 256M\n"
	after := record(t, dir, ctx)

	diff := diffStacks(before, after)
	if !reflect.DeepEqual(diff.Restart, []string{"db"}) || len(diff.Recreate) != 0 {
		t.Errorf("a my.cnf edit produced %+v", diff)
	}
}

// A Dockerfile is the image: the container has to be recreated, and only it.
func TestPhpDockerfileEditRecreatesPhpOnly(t *testing.T) {
	dir := t.TempDir()
	before := record(t, dir, baseCtx())

	ctx := baseCtx()
	ctx["php.Dockerfile"] = "FROM php:8.5-fpm\nRUN apt-get install -y curl\n"
	after := record(t, dir, ctx)

	diff := diffStacks(before, after)
	want := StackDiff{Known: true, Recreate: []string{"php"}}
	if !reflect.DeepEqual(diff, want) {
		t.Errorf("a php Dockerfile edit produced %+v, want %+v", diff, want)
	}
}

// A change to the service's own block — its image tag, an environment
// variable, a port — is a recreate of that service.
func TestServiceBlockChangeRecreatesThatService(t *testing.T) {
	dir := t.TempDir()
	before := record(t, dir, baseCtx())

	var stack map[string]any
	if err := json.Unmarshal(testStack(t, dir), &stack); err != nil {
		t.Fatal(err)
	}
	stack["services"].(map[string]any)["db"].(map[string]any)["image"] = "mariadb:11.8"
	data, _ := json.Marshal(stack)
	after, err := recordFromComposeConfig(data, dir)
	if err != nil {
		t.Fatal(err)
	}

	diff := diffStacks(before, after)
	if !reflect.DeepEqual(diff.Recreate, []string{"db"}) || len(diff.Restart) != 0 || diff.Global {
		t.Errorf("an image change produced %+v", diff)
	}
}

// Anything outside `services` is everyone's: a network renamed is a stack
// recreated, and Narrow must say no.
func TestNetworkChangeIsGlobal(t *testing.T) {
	dir := t.TempDir()
	before := record(t, dir, baseCtx())

	var stack map[string]any
	if err := json.Unmarshal(testStack(t, dir), &stack); err != nil {
		t.Fatal(err)
	}
	stack["networks"].(map[string]any)["isolated"] = map[string]any{"internal": true}
	data, _ := json.Marshal(stack)
	after, err := recordFromComposeConfig(data, dir)
	if err != nil {
		t.Fatal(err)
	}

	diff := diffStacks(before, after)
	if !diff.Global || diff.Narrow() {
		t.Errorf("a network change was narrowed: %+v", diff)
	}
}

// A service that appears is created; one that disappears cannot be narrowed —
// `up` does not remove, and the full rebuild is what does.
func TestServicesAddedAndRemoved(t *testing.T) {
	dir := t.TempDir()
	before := record(t, dir, baseCtx())

	var stack map[string]any
	if err := json.Unmarshal(testStack(t, dir), &stack); err != nil {
		t.Fatal(err)
	}
	services := stack["services"].(map[string]any)
	services["redis"] = map[string]any{"image": "redis:7"}
	data, _ := json.Marshal(stack)
	added, err := recordFromComposeConfig(data, dir)
	if err != nil {
		t.Fatal(err)
	}
	if diff := diffStacks(before, added); !reflect.DeepEqual(diff.Recreate, []string{"redis"}) || !diff.Narrow() {
		t.Errorf("a new service produced %+v", diff)
	}

	delete(services, "redis")
	delete(services, "db")
	data, _ = json.Marshal(stack)
	removed, err := recordFromComposeConfig(data, dir)
	if err != nil {
		t.Fatal(err)
	}
	if diff := diffStacks(before, removed); !reflect.DeepEqual(diff.Removed, []string{"db"}) || diff.Narrow() {
		t.Errorf("a removed service produced %+v", diff)
	}
}

// The application's own files are mounted into every container. They change
// on every edit and are not a reason to touch a container.
func TestSourceTreeIsNotPartOfTheRecord(t *testing.T) {
	dir := t.TempDir()
	before := record(t, dir, baseCtx())

	if err := os.WriteFile(filepath.Join(dir, "src", "index.php"), []byte("<?php echo 1;"), 0o644); err != nil {
		t.Fatal(err)
	}
	after := record(t, dir, baseCtx())

	if diff := diffStacks(before, after); !diff.Empty() {
		t.Errorf("an edit in the source tree produced %+v", diff)
	}
}

// What a Dockerfile copies in is part of the image. `--from` copies from
// another stage and a URL is not a file; both are left alone.
func TestCopiedSources(t *testing.T) {
	dockerfile := `FROM node:22 AS build
COPY package.json package-lock.json /app/
ADD https://example.com/tool.tgz /opt/
COPY --chown=1000:1000 scripts/ /usr/local/bin/
COPY --from=build /app/dist /srv/
RUN echo done
`
	got := copiedSources(dockerfile)
	want := []string{"package.json", "package-lock.json", "scripts/"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("copiedSources = %v, want %v", got, want)
	}
}

// A copied file changing is an image change, even though the Dockerfile text
// did not move.
func TestCopiedFileChangeRecreates(t *testing.T) {
	dir := t.TempDir()
	ctx := baseCtx()
	ctx["php.Dockerfile"] = "FROM php:8.4-fpm\nCOPY scripts/entry.sh /usr/local/bin/\n"
	ctx["scripts/entry.sh"] = "#!/bin/sh\nexec php-fpm\n"
	before := record(t, dir, ctx)

	ctx["scripts/entry.sh"] = "#!/bin/sh\nset -e\nexec php-fpm\n"
	after := record(t, dir, ctx)

	if diff := diffStacks(before, after); !reflect.DeepEqual(diff.Recreate, []string{"php"}) {
		t.Errorf("a change to a copied file produced %+v", diff)
	}
}

// No record from a previous run is not "everything changed" and not "nothing
// changed" — it is no answer, and the caller falls back to what it always did.
func TestDiffWithoutARecordIsUnknown(t *testing.T) {
	real := composeConfigJSON
	composeConfigJSON = func(string) ([]byte, error) { return testStack(t, t.TempDir()), nil }
	t.Cleanup(func() { composeConfigJSON = real })

	diff := DiffStack("no-such-project-" + t.Name())
	if diff.Known {
		t.Errorf("a project with no record answered %+v", diff)
	}
	if diff.Narrow() || diff.Empty() {
		t.Error("an unknown diff must be neither narrow nor empty")
	}
}
