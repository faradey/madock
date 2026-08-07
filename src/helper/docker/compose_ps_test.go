package docker

import "testing"

// Newer compose writes one JSON object per line.
func TestParseComposePSNDJSON(t *testing.T) {
	out := []byte(`{"Service":"php","Name":"madock_demo-php-1","State":"running","ExitCode":0}
{"Service":"nodejs","Name":"madock_demo-nodejs-1","State":"exited","ExitCode":1}`)

	entries := parseComposePS(out)
	if len(entries) != 2 {
		t.Fatalf("parsed %d entries, want 2", len(entries))
	}
	if entries[1].Service != "nodejs" || entries[1].State != "exited" || entries[1].ExitCode != 1 {
		t.Errorf("second entry = %+v, want the exited nodejs service with exit 1", entries[1])
	}
}

// Older compose writes a single JSON array.
func TestParseComposePSArray(t *testing.T) {
	out := []byte(`[{"Service":"php","State":"running","ExitCode":0},{"Service":"nodejs","State":"exited","ExitCode":137}]`)

	entries := parseComposePS(out)
	if len(entries) != 2 {
		t.Fatalf("parsed %d entries, want 2", len(entries))
	}
	if entries[1].ExitCode != 137 {
		t.Errorf("exit code = %d, want 137", entries[1].ExitCode)
	}
}

// Anything that is not a container row must not become one — a warning about a
// service that does not exist is worse than no warning.
func TestParseComposePSIgnoresNoise(t *testing.T) {
	out := []byte("\n  \nnot json\n{\"State\":\"running\"}\n")

	if entries := parseComposePS(out); len(entries) != 0 {
		t.Errorf("parsed %+v, want nothing: blank lines, prose and a row with no service", entries)
	}
}
