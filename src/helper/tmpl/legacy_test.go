package tmpl

import (
	"strings"
	"testing"
)

func convertLegacy(t *testing.T, body string) string {
	t.Helper()
	out, notes := Legacy(body)
	for _, note := range notes {
		t.Logf("note: %s", note)
	}
	return out
}

// docker/snippets/docker-compose/db.yml, which carries every awkward shape the
// tree has: a tag owning its line, a tag with content behind it on the same
// line, an <<<endif>>> with content behind it, an <<<endif>>> attached to the
// end of a content line, and two of the fake config keys.
func TestConvertsTheAwkwardShapes(t *testing.T) {
	body := strings.Join([]string{
		`<<<if{{{db/enabled}}}{{{db/type_is_mysql}}}>>>`,
		`  db:`,
		`    init: true`,
		`<<<if{{{db/use_default_auth_plugin}}}>>>    command:`,
		`      --default-authentication-plugin=mysql_native_password`,
		`<<<endif>>>    build:`,
		`      context: ctx`,
		`    ports:`,
		`      - "{{{port/db}}}:3306"`,
		`    <<<if{{{isolation/enabled}}}>>>networks:`,
		`      - isolated<<<endif>>>`,
		`    restart: {{{restart_policy}}}`,
		`<<<endif>>>`,
		``,
	}, "\n")

	want := strings.Join([]string{
		`{{{- if and .db.enabled (eq .db.type "mysql")}}}`,
		`  db:`,
		`    init: true`,
		`    {{{- if (not (and (eq (lower .db.repository) "mysql") (versionGte .db.version "8.4")))}}}`,
		`    command:`,
		`      --default-authentication-plugin=mysql_native_password`,
		`    {{{- end}}}`,
		`    build:`,
		`      context: ctx`,
		`    ports:`,
		`      - "{{{port "db"}}}:3306"`,
		`    {{{- if .isolation.enabled}}}`,
		`    networks:`,
		`      - isolated`,
		`    {{{- end}}}`,
		`    restart: {{{.restart_policy}}}`,
		`{{{- end}}}`,
	}, "\n")

	if got := convertLegacy(t, body); got != want {
		t.Errorf("converted:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestConvertsIncludes(t *testing.T) {
	got := convertLegacy(t, `{{{include snippets/docker-compose/php.yml}}}`)
	want := `{{{template "snippets/docker-compose/php.yml" .}}}`
	if got != want {
		t.Errorf("converted %q, want %q", got, want)
	}
}

// An include on an indented line takes the indentation with it. Every compose
// file is a column of thirty of these, and the two spaces belong to nothing —
// the snippet carries its own. Left behind, a switched-off service leaves a
// line of trailing whitespace in the generated file.
func TestConvertsIndentedIncludes(t *testing.T) {
	got := convertLegacy(t, "services:\n  {{{include snippets/docker-compose/php.yml}}}\n")
	want := "services:\n  {{{- template \"snippets/docker-compose/php.yml\" .}}}\n"
	if got != want {
		t.Errorf("converted %q, want %q", got, want)
	}
}

// docker/snippets/docker-compose/volumes.yml: a whole conditional volume entry
// on one line, with the entry's own indentation sitting in front of the tag
// rather than in front of the entry. Splitting that into three lines naively
// puts the volume at column zero, which compose reads as a different mapping —
// a whole volumes block silently misplaced, and valid YAML either way.
func TestConvertsAWholeBlockOnOneLine(t *testing.T) {
	got := convertLegacy(t, "  dbdata:\n  <<<if{{{db2/enabled}}}>>>dbdata2:<<<endif>>>\n")
	want := "  dbdata:\n" +
		"  {{{- if .db2.enabled}}}\n" +
		"  dbdata2:\n" +
		"  {{{- end}}}\n"
	if got != want {
		t.Errorf("converted:\n%q\nwant:\n%q", got, want)
	}
}

// A file whose last line is a conditional loses its final newline to the trim
// marker, and the compose file it was included into ended mid-line.
func TestFileWithoutATrailingNewlineGetsOne(t *testing.T) {
	got := convertLegacy(t, "networks:\n  madock-proxy:\n<<<if{{{isolation/enabled}}}>>>\n  isolated:\n<<<endif>>>")
	if !strings.HasSuffix(got, "{{{- end}}}\n") {
		t.Errorf("converted %q, want it to end with a newline", got)
	}
}

func TestConvertsElse(t *testing.T) {
	body := strings.Join([]string{
		`    <<<if{{{php/xdebug/enabled}}}>>>`,
		`    with-xdebug`,
		`    <<<else>>>`,
		`    without-xdebug`,
		`    <<<endif>>>`,
	}, "\n")

	want := strings.Join([]string{
		`    {{{- if .php.xdebug.enabled}}}`,
		`    with-xdebug`,
		`    {{{- else}}}`,
		`    without-xdebug`,
		`    {{{- end}}}`,
	}, "\n")

	if got := convertLegacy(t, body); got != want {
		t.Errorf("converted:\n%s\nwant:\n%s", got, want)
	}
}

// The joined lists. Their separator carried the indentation of the file they
// were going into — "\n      " — which is how a compose file's shape came to be
// decided in Go source.
func TestConvertsTheJoinedLists(t *testing.T) {
	got := convertLegacy(t, "    extra_hosts:\n      {{{nginx/host_gateways}}}\n")
	want := "    extra_hosts:\n" +
		"      {{{- range .nginx.hosts}}}\n" +
		"      - \"{{{.name}}}:host-gateway\"\n" +
		"      {{{- end}}}\n"
	if got != want {
		t.Errorf("converted:\n%q\nwant:\n%q", got, want)
	}

	got = convertLegacy(t, "    server_name {{{nginx/host_names}}};")
	want = `    server_name {{{range $i, $host := .nginx.hosts}}}{{{if $i}}} {{{end}}}{{{$host.name}}}{{{end}}};`
	if got != want {
		t.Errorf("converted %q, want %q", got, want)
	}
}

// A here-string is not a tag, and three Dockerfiles contain one. The tag regex
// has to leave it alone or the delimiter argument was for nothing.
func TestHereStringIsNotATag(t *testing.T) {
	got := convertLegacy(t, `IFS='.' read major minor patch <<< "{{{php/version}}}"`)
	want := `IFS='.' read major minor patch <<< "{{{.php.version}}}"`
	if got != want {
		t.Errorf("converted %q, want %q", got, want)
	}
}

// Grafana dashboards pass through the engine and are full of {{queue}} and
// {{.State.Status}}. Two braces are not three.
func TestTwoBracesAreLeftAlone(t *testing.T) {
	body := `"expr": "rate({job=\"x\"}[5m])", "legendFormat": "{{queue}} {{ command }}"`
	if got := convertLegacy(t, body); got != body {
		t.Errorf("converted %q, want it untouched", got)
	}
}

func TestIsLegacy(t *testing.T) {
	legacy := []string{
		`<<<if{{{php/enabled}}}>>>x<<<endif>>>`,
		`{{{php/version}}}`,
		`{{{restart_policy}}}`,
		`{{{include snippets/docker-compose/php.yml}}}`,
	}
	for _, body := range legacy {
		if !IsLegacy(body) {
			t.Errorf("%q was not recognised as the old syntax", body)
		}
	}

	converted := []string{
		`{{{.php.version}}}`,
		`{{{- if .php.enabled}}}x{{{- end}}}`,
		`{{{template "snippets/docker-compose/php.yml" .}}}`,
		`{{{port "livereload"}}}`,
		`{{{range $i, $host := .nginx.hosts}}}{{{$host.name}}}{{{end}}}`,
		`{{{(index .nginx.hosts 0).name}}}`,
		`"legendFormat": "{{queue}}"`,
		`docker inspect --format '{{.State.Status}}'`,
		`awk '{print $1}'`,
	}
	for _, body := range converted {
		if IsLegacy(body) {
			t.Errorf("%q was mistaken for the old syntax", body)
		}
	}
}

// A project's own override under .madock/docker/ keeps working, and says so.
func TestLegacyOverrideIsConvertedAtRenderTime(t *testing.T) {
	var warnedAbout string
	var warnedNotes []string

	r := &Renderer{
		Values:   map[string]string{"php/enabled": "true", "php/version": "8.4"},
		OnLegacy: func(name string, notes []string) { warnedAbout, warnedNotes = name, notes },
	}

	got, err := r.Render(".madock/docker/php/Dockerfile", "<<<if{{{php/enabled}}}>>>\nFROM php:{{{php/version}}}\n<<<endif>>>\n")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// The trailing newline is the snippet's, not the conditional's, and a file
	// that is one conditional from top to bottom renders to nothing at all when
	// it is off — newline included.
	if got != "\nFROM php:8.4" {
		t.Errorf("rendered %q", got)
	}
	if warnedAbout != ".madock/docker/php/Dockerfile" {
		t.Errorf("the warning named %q", warnedAbout)
	}
	if len(warnedNotes) != 0 {
		t.Errorf("unexpected notes: %v", warnedNotes)
	}
}

func TestUnbalancedTagIsReported(t *testing.T) {
	_, notes := Legacy("<<<if{{{php/enabled}}}>>>\nnever closed\n")
	if len(notes) == 0 {
		t.Fatal("an unclosed <<<if was converted without a word")
	}
	if !strings.Contains(notes[0], "without an <<<endif>>>") {
		t.Errorf("unexpected note: %q", notes[0])
	}
}
