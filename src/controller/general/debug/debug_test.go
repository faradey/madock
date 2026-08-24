package debug

import "testing"

// One command, and the project decides what it means. The alternative was a
// single `debug/enabled` derived from `language`, and this is the case that
// rules it out: a PHP application with a JavaScript front end in its own
// container runs two debuggable things at once, and one switch would have had
// to pick a winner silently.
func TestDebuggableRuntimes(t *testing.T) {
	tests := []struct {
		name string
		conf map[string]string
		want []string
	}{
		{
			name: "php alone is what it has always been",
			conf: map[string]string{"php/enabled": "true"},
			want: []string{"php/xdebug/enabled"},
		},
		{
			name: "node alone",
			conf: map[string]string{"nodejs/enabled": "true"},
			want: []string{"nodejs/debug/enabled"},
		},
		{
			name: "both, and neither is dropped",
			conf: map[string]string{"php/enabled": "true", "nodejs/enabled": "true"},
			want: []string{"php/xdebug/enabled", "nodejs/debug/enabled"},
		},
		{
			// Node inside the application container has no container of its own,
			// so nothing can publish the port its debugger needs. Counting it
			// would turn on a switch that renders nothing.
			name: "node embedded in another container is not debuggable",
			conf: map[string]string{"nodejs/embedded/enabled": "true"},
			want: nil,
		},
		{
			name: "a project with neither",
			conf: map[string]string{"language": "golang"},
			want: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			found := debuggable(test.conf)

			if len(found) != len(test.want) {
				t.Fatalf("got %d runtimes, want %d: %+v", len(found), len(test.want), found)
			}
			for i, key := range test.want {
				if found[i].key != key {
					t.Errorf("runtime %d writes %q, want %q", i, found[i].key, key)
				}
				if found[i].name == "" {
					t.Errorf("runtime %d has no name, so the output cannot say what it turned on", i)
				}
			}
		})
	}
}
