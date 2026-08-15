package tmpl

import (
	"strings"
	"testing"
)

func render(t *testing.T, body string, values map[string]string) string {
	t.Helper()
	r := &Renderer{Values: values}
	out, err := r.Render("test", body)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return out
}

// The one silent regression the rewrite makes available: config values are
// strings, and every non-empty string is true in a Go template. "false" has to
// arrive as a bool or every conditional in madock fires.
func TestFalseIsFalsy(t *testing.T) {
	got := render(t, `{{{if .php.enabled}}}on{{{else}}}off{{{end}}}`, map[string]string{
		"php/enabled": "false",
	})
	if got != "off" {
		t.Errorf(`php/enabled="false" rendered %q, want "off"`, got)
	}
}

func TestTrueIsTruthy(t *testing.T) {
	got := render(t, `{{{if .php.enabled}}}on{{{else}}}off{{{end}}}`, map[string]string{
		"php/enabled": "true",
	})
	if got != "on" {
		t.Errorf(`php/enabled="true" rendered %q, want "on"`, got)
	}
}

// The old engine read a condition by searching the text for "false", so any
// value that happened to contain the word flipped the branch. This is the
// defect the rewrite exists to remove, so it is pinned.
func TestValueContainingFalseIsNotFalsy(t *testing.T) {
	got := render(t, `{{{if .varnish.config_file}}}on{{{else}}}off{{{end}}}`, map[string]string{
		"varnish/config_file": "/etc/false-positive.vcl",
	})
	if got != "on" {
		t.Errorf(`a path containing "false" rendered %q, want "on"`, got)
	}
}

// A shared snippet asks about memcached on a platform whose config has never
// heard of it. The old engine left the placeholder standing and read it as
// false; making it a render failure would break `madock start` everywhere.
func TestKeyTheProjectDoesNotHaveIsFalsy(t *testing.T) {
	got := render(t, `{{{if .memcached.enabled}}}on{{{else}}}off{{{end}}}`, map[string]string{
		"php/enabled": "true",
	})
	if got != "off" {
		t.Errorf(`an absent key rendered %q, want "off"`, got)
	}
}

func TestAbsentKeyPrintsNothing(t *testing.T) {
	got := render(t, `[{{{.memcached.memory}}}]`, map[string]string{})
	if got != "[]" {
		t.Errorf("an absent key printed %q, want %q", got, "[]")
	}
}

// A version is a string and must stay one, or versionLt has nothing to compare.
func TestVersionComparison(t *testing.T) {
	body := `{{{if and (eq .db.type "mysql") (versionLt .db.version "8.4")}}}legacy-auth{{{end}}}`

	got := render(t, body, map[string]string{"db/type": "mysql", "db/version": "8.0.36"})
	if got != "legacy-auth" {
		t.Errorf("mysql 8.0.36 rendered %q, want %q", got, "legacy-auth")
	}

	got = render(t, body, map[string]string{"db/type": "mysql", "db/version": "8.4"})
	if got != "" {
		t.Errorf("mysql 8.4 rendered %q, want empty", got)
	}

	got = render(t, body, map[string]string{"db/type": "mariadb", "db/version": "10.6"})
	if got != "" {
		t.Errorf("mariadb rendered %q, want empty", got)
	}
}

// Hosts arrive as ordinary config keys — nginx/hosts/<code>/name — so the tree
// alone already gives a template the loop the old engine did not have. Go
// iterates a map in sorted key order, which is the order GetHosts produced by
// sorting too.
func TestHostsRangeInsteadOfAPreJoinedString(t *testing.T) {
	got := render(t, `{{{range $code, $host := .nginx.hosts}}}{{{$host.name}}} {{{$code}}};{{{end}}}`, map[string]string{
		"nginx/hosts/base/name":  "shop.test",
		"nginx/hosts/base2/name": "second.test",
	})
	want := "shop.test base;second.test base2;"
	if got != want {
		t.Errorf("hosts rendered %q, want %q", got, want)
	}
}

// A map is not enough where order matters — "the first host" is a real question
// the nginx and grafana templates ask — so the caller replaces it with a slice.
// The indentation in the loop body is the point: it used to live in Go, as the
// separator of a strings.Join.
func TestDataReplacesConfigKeysWithAnOrderedList(t *testing.T) {
	r := &Renderer{
		Values: map[string]string{"nginx/hosts/base/name": "shop.test"},
		Data: map[string]any{
			"nginx/hosts": []map[string]string{
				{"name": "shop.test", "code": "base"},
				{"name": "second.test", "code": "base2"},
			},
		},
	}

	body := "    extra_hosts:\n" +
		"      - \"host.docker.internal:host-gateway\"\n" +
		"      {{{- range .nginx.hosts}}}\n" +
		"      - \"{{{.name}}}:host-gateway\"\n" +
		"      {{{- end}}}\n" +
		"    ports: []\n"

	got, err := r.Render("test", body)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "    extra_hosts:\n" +
		"      - \"host.docker.internal:host-gateway\"\n" +
		"      - \"shop.test:host-gateway\"\n" +
		"      - \"second.test:host-gateway\"\n" +
		"    ports: []\n"
	if got != want {
		t.Errorf("rendered:\n%q\nwant:\n%q", got, want)
	}

	first, err := r.Render("first", `{{{(index .nginx.hosts 0).name}}}`)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if first != "shop.test" {
		t.Errorf("the first host rendered %q", first)
	}
}

// No hosts configured means no lines, not a line of trailing whitespace — which
// is what an empty strings.Join left behind.
func TestEmptyHostListLeavesNoLine(t *testing.T) {
	r := &Renderer{
		Values: map[string]string{},
		Data:   map[string]any{"nginx/hosts": []map[string]string{}},
	}
	got, err := r.Render("test", "a:\n  {{{- range .nginx.hosts}}}\n  - \"{{{.name}}}\"\n  {{{- end}}}\nb:\n")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "a:\nb:\n" {
		t.Errorf("rendered %q, want %q", got, "a:\nb:\n")
	}
}

func TestIncludeResolvesRecursively(t *testing.T) {
	snippets := map[string]string{
		"snippets/outer": `outer({{{template "snippets/inner" .}}})`,
		"snippets/inner": `inner:{{{.php.version}}}`,
	}
	r := &Renderer{
		Values:  map[string]string{"php/version": "8.4"},
		Snippet: func(name string) (string, error) { return snippets[name], nil },
	}

	got, err := r.Render("root", `[{{{template "snippets/outer" .}}}]`)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "[outer(inner:8.4)]" {
		t.Errorf("rendered %q", got)
	}
}

// The old include pass was a regex looped "while a match remains" and had no
// cycle detection at all: a snippet including itself spun forever. Loading a
// name once means the load terminates; a genuinely recursive template is then
// text/template's problem and it stops with a named error.
func TestSelfIncludingSnippetDoesNotHang(t *testing.T) {
	r := &Renderer{
		Values:  map[string]string{},
		Snippet: func(name string) (string, error) { return `loop{{{template "snippets/loop" .}}}`, nil },
	}

	_, err := r.Render("root", `{{{template "snippets/loop" .}}}`)
	if err == nil {
		t.Fatal("a self-including snippet rendered without an error")
	}
	if !strings.Contains(err.Error(), "exceeded maximum template depth") {
		t.Errorf("unexpected error for a template cycle: %v", err)
	}
}

// Resolving a port allocates it. The function form says so; the old
// {{{port/livereload}}} placeholder looked like configuration and was not.
func TestPortIsAFunctionCall(t *testing.T) {
	asked := ""
	r := &Renderer{
		Values: map[string]string{},
		Port: func(service string) (int, error) {
			asked = service
			return 35729, nil
		},
	}

	got, err := r.Render("test", `- "{{{port "livereload"}}}:35729"`)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != `- "35729:35729"` {
		t.Errorf("rendered %q", got)
	}
	if asked != "livereload" {
		t.Errorf("allocator was asked for %q, want %q", asked, "livereload")
	}
}

// Weakness six of the old engine: an error with no address. It reported a line
// number in the rendered output, not in the source, and only for a leftover
// <<< tag — a stray {{{placeholder}}} was not caught at all, which is how
// literal placeholders reached a generated compose file and broke every project
// at once.
func TestUnbalancedTagIsAnErrorAndNamesTheTemplate(t *testing.T) {
	r := &Renderer{Values: map[string]string{}}
	_, err := r.Render("docker-compose/php.yml", "line one\nline two\n{{{if .php.enabled}}}\nnever closed\n")
	if err == nil {
		t.Fatal("an unclosed if parsed without an error")
	}
	if !strings.Contains(err.Error(), "docker-compose/php.yml") {
		t.Errorf("error does not name the template: %v", err)
	}
	// The old engine wrote the file out anyway, every conditional unresolved,
	// which produced an nginx config with six server blocks where one belonged.
	// A test in docker/ counted tags for exactly this reason; the parser does it
	// now.
	if !strings.Contains(err.Error(), "unexpected EOF") {
		t.Errorf("unexpected error for an unclosed if: %v", err)
	}
}

func TestErrorNamesTheLine(t *testing.T) {
	r := &Renderer{Values: map[string]string{}}
	_, err := r.Render("docker-compose/php.yml", "line one\nline two\n{{{nosuchfunc .php.enabled}}}\n")
	if err == nil {
		t.Fatal("an unknown function parsed without an error")
	}
	if !strings.Contains(err.Error(), "docker-compose/php.yml:3") {
		t.Errorf("error does not address the template and line: %v", err)
	}
}

func TestKeyThatIsBothValueAndGroupIsReported(t *testing.T) {
	r := &Renderer{Values: map[string]string{
		"db/type":   "mysql",
		"db/type/x": "no",
	}}
	_, err := r.Render("test", "x")
	if err == nil {
		t.Fatal("a key that is both a value and a group rendered without an error")
	}
	if !strings.Contains(err.Error(), "db/type") {
		t.Errorf("error does not name the key: %v", err)
	}
}

// Whitespace is the half of this migration that has no shortcut, so the shape
// the converter emits is pinned here: the trim markers have to delete the line
// a conditional sat on, exactly as the old engine's hand-rolled cleanup did.
func TestTrimMarkersRemoveTheLineAConditionalSatOn(t *testing.T) {
	body := "services:\n" +
		"  php:\n" +
		"    {{{- if .isolation.enabled}}}\n" +
		"    networks:\n" +
		"      - isolated\n" +
		"    {{{- end}}}\n" +
		"    restart: {{{.restart_policy}}}\n"

	got := render(t, body, map[string]string{"isolation/enabled": "false", "restart_policy": "no"})
	want := "services:\n  php:\n    restart: no\n"
	if got != want {
		t.Errorf("disabled block rendered:\n%q\nwant:\n%q", got, want)
	}

	got = render(t, body, map[string]string{"isolation/enabled": "true", "restart_policy": "no"})
	want = "services:\n  php:\n    networks:\n      - isolated\n    restart: no\n"
	if got != want {
		t.Errorf("enabled block rendered:\n%q\nwant:\n%q", got, want)
	}
}

// A bash here-string is legal template content, and three Dockerfile templates
// contain one. It is the reason the delimiters stay {{{ }}} rather than becoming
// <<< >>>, so it is worth a test rather than a note.
func TestHereStringSurvives(t *testing.T) {
	got := render(t, `IFS='.' read major minor patch <<< "{{{.php.version}}}"`, map[string]string{"php/version": "8.4.1"})
	if got != `IFS='.' read major minor patch <<< "8.4.1"` {
		t.Errorf("rendered %q", got)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"8.4", "8.3.1", 1},
		{"8.3.1", "8.4", -1},
		{"8.4", "8.4.0", 0},
		{"10.6", "9.9", 1},
		{"", "0", 0},
	}
	for _, c := range cases {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
