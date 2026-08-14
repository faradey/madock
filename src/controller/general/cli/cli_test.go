package cli

import (
	"reflect"
	"testing"
)

// TestTakeServiceFlag covers the one thing that makes --service safe to add to a
// command whose remaining arguments belong to somebody else: everything after the
// flag has to survive untouched, flags of its own included. A parser that owned the
// whole line is why `madock bash --service php -c '…'` answers "unknown argument
// -c" and why there was no way to run a command in a service other than the main
// one.
func TestTakeServiceFlag(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantService string
		wantRest    []string
	}{
		{
			name:     "no flag",
			args:     []string{"php", "-v"},
			wantRest: []string{"php", "-v"},
		},
		{
			name:        "--service with a value",
			args:        []string{"--service", "php", "php", "-v"},
			wantService: "php",
			wantRest:    []string{"php", "-v"},
		},
		{
			name:        "--service=value",
			args:        []string{"--service=nodejs", "npm", "run", "build"},
			wantService: "nodejs",
			wantRest:    []string{"npm", "run", "build"},
		},
		{
			name:        "short form",
			args:        []string{"-s", "db", "mysql", "--version"},
			wantService: "db",
			wantRest:    []string{"mysql", "--version"},
		},
		{
			name:        "short form with =",
			args:        []string{"-s=ruby", "ruby", "-e", "puts 1"},
			wantService: "ruby",
			wantRest:    []string{"ruby", "-e", "puts 1"},
		},
		{
			// Only the first position is ours. Later on it is the container
			// command's argument and must be passed through, not eaten.
			name:     "flag-looking argument further along",
			args:     []string{"artisan", "queue:work", "--service=redis"},
			wantRest: []string{"artisan", "queue:work", "--service=redis"},
		},
		{
			name:     "nothing at all",
			args:     nil,
			wantRest: nil,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			service, rest := takeServiceFlag(testCase.args)
			if service != testCase.wantService {
				t.Errorf("service = %q, want %q", service, testCase.wantService)
			}
			if !reflect.DeepEqual(rest, testCase.wantRest) {
				t.Errorf("rest = %q, want %q", rest, testCase.wantRest)
			}
		})
	}
}
