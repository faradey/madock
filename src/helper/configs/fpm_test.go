package configs

import (
	"strings"
	"testing"
)

// The four pool values are independent settings and not independent numbers.
// php-fpm checks them against each other at start-up, so a disagreement is
// answered from inside the container after a full image rebuild — on a project
// that was working ten minutes ago.
func TestValidateFpmPool(t *testing.T) {
	pool := func(children, start, minSpare, maxSpare string) map[string]string {
		return map[string]string{
			"php/fpm/max_children":      children,
			"php/fpm/start_servers":     start,
			"php/fpm/min_spare_servers": minSpare,
			"php/fpm/max_spare_servers": maxSpare,
		}
	}

	tests := []struct {
		name    string
		conf    map[string]string
		wantErr string
	}{
		{
			name: "the shipped defaults",
			conf: pool("40", "2", "1", "3"),
		},
		{
			name: "the pre-warmed pool somebody may prefer",
			conf: pool("40", "8", "4", "16"),
		},
		{
			// The case that reaches php-fpm today: only the cap is lowered, and
			// the spare bounds are left where the defaults put them.
			name:    "cap lowered below the spare ceiling",
			conf:    pool("2", "2", "1", "3"),
			wantErr: "more spare workers than workers",
		},
		{
			name:    "floor above the ceiling",
			conf:    pool("40", "5", "9", "4"),
			wantErr: "floor is above",
		},
		{
			name:    "start outside the spare band",
			conf:    pool("40", "12", "1", "3"),
			wantErr: "outside",
		},
		{
			name:    "zero workers",
			conf:    pool("0", "2", "1", "3"),
			wantErr: "at least one",
		},
		{
			name:    "not a number",
			conf:    pool("many", "2", "1", "3"),
			wantErr: "not a number",
		},
		{
			// Half a pool is answered by the embedded defaults for the rest, and
			// comparing against numbers this installation may not be using would
			// refuse a configuration that works.
			name: "only one key set",
			conf: map[string]string{"php/fpm/max_children": "80"},
		},
		{
			name: "nothing set at all",
			conf: map[string]string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateFpmPool(test.conf)

			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("a pool php-fpm would accept was refused: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("a pool php-fpm refuses to start with was accepted")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("the refusal does not say %q: %v", test.wantErr, err)
			}
		})
	}
}

// The numbers are never adjusted into something valid: a pool quietly changed to
// what madock thinks the author meant is a configuration that says one thing and
// runs another.
func TestValidateFpmPoolDoesNotRewriteTheConfig(t *testing.T) {
	conf := map[string]string{
		"php/fpm/max_children":      "2",
		"php/fpm/start_servers":     "2",
		"php/fpm/min_spare_servers": "1",
		"php/fpm/max_spare_servers": "3",
	}

	_ = ValidateFpmPool(conf)

	if conf["php/fpm/max_spare_servers"] != "3" || conf["php/fpm/max_children"] != "2" {
		t.Errorf("the configuration was modified: %v", conf)
	}
}
