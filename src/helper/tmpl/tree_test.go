package tmpl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tree conversion is what a user with an override actually runs, through
// `madock template:convert`. Everything below is about that: it has to rewrite
// the file, say what it did, leave a second run alone, and touch nothing when
// asked not to.
func TestConvertTree(t *testing.T) {
	legacy := "<<<if{{{php/enabled}}}>>>\n  php:\n    image: php:{{{php/version}}}\n<<<endif>>>\n"

	t.Run("rewrites the file and names it", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "docker-compose.yml")
		write(t, path, legacy)

		report, err := ConvertTree(dir, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Converted) != 1 || report.Converted[0] != path {
			t.Fatalf("converted %v, want [%s]", report.Converted, path)
		}
		if len(report.Notes) != 0 {
			t.Errorf("unexpected notes: %v", report.Notes)
		}

		got := read(t, path)
		if IsLegacy(got) {
			t.Errorf("the file is still in the old syntax:\n%s", got)
		}
		if !strings.Contains(got, "{{{- if .php.enabled}}}") {
			t.Errorf("converted to:\n%s", got)
		}
	})

	// A rebase is "take theirs, run the converter again", so a second run has to
	// be a no-op rather than a second conversion of already-converted text.
	t.Run("a second run changes nothing", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "docker-compose.yml")
		write(t, path, legacy)

		if _, err := ConvertTree(dir, false); err != nil {
			t.Fatal(err)
		}
		once := read(t, path)

		report, err := ConvertTree(dir, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Converted) != 0 {
			t.Errorf("a second run converted %v", report.Converted)
		}
		if got := read(t, path); got != once {
			t.Errorf("a second run rewrote the file:\n%s", got)
		}
	})

	t.Run("dry run writes nothing", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "docker-compose.yml")
		write(t, path, legacy)

		report, err := ConvertTree(dir, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Converted) != 1 {
			t.Fatalf("dry run reported %d files, want 1", len(report.Converted))
		}
		if got := read(t, path); got != legacy {
			t.Errorf("dry run rewrote the file:\n%s", got)
		}
	})

	// A template that does not parse after conversion is reported rather than
	// written out to be discovered at the next `madock start`.
	t.Run("says when the result does not parse", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "broken.yml"), "<<<if{{{php/enabled}}}>>>\nnever closed\n")

		report, err := ConvertTree(dir, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Notes) == 0 {
			t.Fatal("an unbalanced conditional was converted without a word")
		}
		if !strings.Contains(strings.Join(report.Notes, "\n"), "broken.yml") {
			t.Errorf("the notes do not name the file: %v", report.Notes)
		}
	})

	t.Run("counts what needed nothing", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "already.yml"), "{{{- if .php.enabled}}}\nx\n{{{- end}}}\n")

		report, err := ConvertTree(dir, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Converted) != 0 || report.Unchanged != 1 {
			t.Fatalf("converted %v, unchanged %d; want none converted and one unchanged", report.Converted, report.Unchanged)
		}
	})
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
