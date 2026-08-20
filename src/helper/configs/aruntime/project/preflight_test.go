package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/tmpl"
)

// checkerFor is the preflight's renderer, pointed at one directory of snippets
// so the test does not need an installation around it.
func checkerFor(t *testing.T, snippetDir string) *tmpl.Renderer {
	t.Helper()

	return &tmpl.Renderer{
		Snippet: func(name string) (string, error) {
			body, err := os.ReadFile(filepath.Join(snippetDir, name))
			if err != nil {
				// The same shape FindSnippetFile returns, which is what the walk
				// filters on.
				return "", errSnippetMissingFor(name)
			}
			return string(body), nil
		},
	}
}

func errSnippetMissingFor(name string) error {
	_, err := FindSnippetFile("no-such-project", name)
	return err
}

func writeTemplate(t *testing.T, path, body string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// An include that no longer resolves is what took two machines down: it was
// found while generating the build context, which happens after the containers
// have been stopped.
func TestBrokenIncludeIsFoundWithoutRendering(t *testing.T) {
	templates := t.TempDir()
	snippets := t.TempDir()

	writeTemplate(t, filepath.Join(templates, "Dockerfile"),
		"FROM php:8.3\n{{{template \"snippets/dockerfile/php/nodejs\"}}}\n")

	problems := brokenIn(templates, checkerFor(t, snippets))
	if len(problems) != 1 {
		t.Fatalf("expected the missing include to be reported once, got %d: %v", len(problems), problems)
	}
	if !strings.Contains(problems[0], "snippets/dockerfile/php/nodejs") {
		t.Errorf("the report does not name the include: %s", problems[0])
	}
	if !strings.Contains(problems[0], "Dockerfile") {
		t.Errorf("the report does not name the file that includes it: %s", problems[0])
	}
}

// A template whose includes resolve is not a problem, and neither is one that
// has no directives at all — a certificate or a script sitting in the same
// directory must not be reported.
func TestResolvableAndPlainFilesAreNotReported(t *testing.T) {
	templates := t.TempDir()
	snippets := t.TempDir()

	writeTemplate(t, filepath.Join(snippets, "snippets/dockerfile/common/nodejs"), "RUN echo node\n")
	writeTemplate(t, filepath.Join(templates, "Dockerfile"),
		"FROM php:8.3\n{{{template \"snippets/dockerfile/common/nodejs\"}}}\n")
	writeTemplate(t, filepath.Join(templates, "ctx/scripts/entrypoint.sh"), "#!/bin/sh\necho hello\n")
	writeTemplate(t, filepath.Join(templates, "ctx/cert.pem"), "-----BEGIN CERTIFICATE-----\n")

	if problems := brokenIn(templates, checkerFor(t, snippets)); len(problems) != 0 {
		t.Fatalf("nothing is broken here, but got: %v", problems)
	}
}

// A template that fails to parse is left to the render, which reports it
// properly. A preflight that also fails on unrelated defects is one people learn
// to skip.
func TestAParseErrorIsNotAPreflightFailure(t *testing.T) {
	templates := t.TempDir()
	snippets := t.TempDir()

	writeTemplate(t, filepath.Join(templates, "Dockerfile"), "FROM php:8.3\n{{{ if .php.enabled }}}\n")

	if problems := brokenIn(templates, checkerFor(t, snippets)); len(problems) != 0 {
		t.Fatalf("a parse error was reported as drift: %v", problems)
	}
}

// The include chain is followed: a snippet that includes a snippet that is gone
// is the same defect one level down, and the level is where the old regex pass
// used to give up.
func TestABrokenIncludeInsideASnippetIsFound(t *testing.T) {
	templates := t.TempDir()
	snippets := t.TempDir()

	writeTemplate(t, filepath.Join(snippets, "snippets/dockerfile/common/php"),
		"{{{template \"snippets/dockerfile/php/nodejs\"}}}\n")
	writeTemplate(t, filepath.Join(templates, "Dockerfile"),
		"FROM php:8.3\n{{{template \"snippets/dockerfile/common/php\"}}}\n")

	problems := brokenIn(templates, checkerFor(t, snippets))
	if len(problems) != 1 {
		t.Fatalf("expected one report, got %d: %v", len(problems), problems)
	}
	if !strings.Contains(problems[0], "php/nodejs") {
		t.Errorf("the report names the wrong include: %s", problems[0])
	}
}

// FindSnippetFile says where it looked, because the answer to "it does not
// exist" is always "where did you expect it".
func TestFindSnippetFileNamesEveryPlaceItLooked(t *testing.T) {
	_, err := FindSnippetFile("e2e", "snippets/dockerfile/php/nodejs")
	if err == nil {
		t.Fatal("a snippet that cannot exist was resolved")
	}
	if !strings.Contains(err.Error(), "Looked in:") || strings.Count(err.Error(), "snippets/dockerfile/php/nodejs") < 2 {
		t.Errorf("the error does not say where it looked: %v", err)
	}
}
