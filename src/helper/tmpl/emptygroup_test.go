package tmpl

import "testing"

// A group that was emptied in a config file is an empty element, and it reads
// back as an empty value next to the keys that still live under the same name.
// That must not stop the render: the empty value is nothing set, so the group
// is what the keys say it is.
func TestRenderReadsAnEmptyValueBesideItsGroupAsTheGroup(t *testing.T) {
	r := &Renderer{Values: map[string]string{
		"varnish":         "",
		"varnish/enabled": "true",
	}}

	got, err := r.Render("test", `{{{if .varnish.enabled}}}on{{{end}}}`)
	if err != nil {
		t.Fatalf("the render failed: %v", err)
	}
	if got != "on" {
		t.Errorf("got %q, want %q", got, "on")
	}
}

// The same value set on its own is still a value, and a group is not made up
// over it.
func TestBuildTreeKeepsANonEmptyValueAGroupCannotReplace(t *testing.T) {
	if _, err := buildTree(map[string]string{
		"varnish":         "x",
		"varnish/enabled": "true",
	}); err == nil {
		t.Error("a value and a group under the same name were both accepted")
	}
}
