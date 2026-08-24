package logs

import (
	"strings"
	"testing"
)

// `madock logs php` used to answer "too many positional arguments at 'php'" —
// an error about the parser, on a command whose help says it shows container
// logs. It takes the name either way now.
func TestChooseService(t *testing.T) {
	tests := []struct {
		name       string
		positional string
		flag       string
		fallback   string
		want       string
		wantErr    bool
	}{
		{
			name:       "the name people type",
			positional: "php",
			fallback:   "nginx",
			want:       "php",
		},
		{
			name:     "the flag every other command uses",
			flag:     "db",
			fallback: "nginx",
			want:     "db",
		},
		{
			name:     "neither, so the platform's main service",
			fallback: "php",
			want:     "php",
		},
		{
			// Same service twice is somebody being explicit, not a mistake.
			name:       "both, agreeing",
			positional: "db",
			flag:       "db",
			fallback:   "nginx",
			want:       "db",
		},
		{
			// Picking one would be a guess, and a guess here comes back as
			// logs: the reader concludes the other service is quiet.
			name:       "both, disagreeing",
			positional: "php",
			flag:       "db",
			fallback:   "nginx",
			wantErr:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := chooseService(test.positional, test.flag, test.fallback)

			if test.wantErr {
				if err == nil {
					t.Fatalf("two different services were resolved to %q instead of being refused", got)
				}
				if !strings.Contains(err.Error(), "php") || !strings.Contains(err.Error(), "db") {
					t.Errorf("the refusal names neither of them: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected refusal: %v", err)
			}
			if got != test.want {
				t.Errorf("service = %q, want %q", got, test.want)
			}
		})
	}
}
