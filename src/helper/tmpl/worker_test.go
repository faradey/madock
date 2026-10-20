package tmpl

import "testing"

// One program is exec'd; anything the shell has to interpret is left to it.
// `exec a && b` never runs b, and `exec set -a; …` fails on exec set — the
// Indemorio engine's workers load their .env exactly that way (IND-11).
func TestAWorkerCommandWithShellSyntaxIsNotExeced(t *testing.T) {
	for cmd, want := range map[string]string{
		"php artisan queue:work redis":                    "exec php artisan queue:work redis",
		"set -a; . ./.env; php artisan queue:work":        "set -a; . ./.env; php artisan queue:work",
		"cd /app && node worker.js":                       "cd /app && node worker.js",
		"node worker.js | tee /var/log/w.log":             "node worker.js | tee /var/log/w.log",
		"php bin/console messenger:consume $(echo async)": "php bin/console messenger:consume $$(echo async)",
		`echo "$GREETING" > /tmp/w`:                       `echo \"$$GREETING\" > /tmp/w`,
		". ./.env && ./run":                               ". ./.env && ./run",
	} {
		if got := workerCommand(cmd); got != want {
			t.Errorf("workerCommand(%q) = %q, want %q", cmd, got, want)
		}
	}
}
