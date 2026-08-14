package project

import "testing"

// TestResolveMainServiceEnabled covers every service the resolver can name.
//
// php and nodejs were answered honestly; python, golang, ruby and app fell through
// to a blanket "true". Each of those is rendered into docker-compose behind its own
// <<<if{{{<service>/enabled}}}>>>, so a "true" for a service that is switched off
// produces a file referring to something it does not contain — a `depends_on` docker
// compose refuses to read, and an nginx upstream on a name that does not resolve.
func TestResolveMainServiceEnabled(t *testing.T) {
	services := []string{"php", "nodejs", "python", "golang", "ruby", "app"}

	for _, service := range services {
		t.Run(service+" enabled", func(t *testing.T) {
			conf := map[string]string{service + "/enabled": "true"}
			if got := resolveMainServiceEnabled(conf, service); got != "true" {
				t.Errorf("%s enabled -> %q, want \"true\"", service, got)
			}
		})

		t.Run(service+" disabled", func(t *testing.T) {
			conf := map[string]string{service + "/enabled": "false"}
			if got := resolveMainServiceEnabled(conf, service); got != "false" {
				t.Errorf("%s disabled -> %q, want \"false\"", service, got)
			}
		})

		t.Run(service+" unset", func(t *testing.T) {
			if got := resolveMainServiceEnabled(map[string]string{}, service); got != "false" {
				t.Errorf("%s unset -> %q, want \"false\"", service, got)
			}
		})
	}

	// A platform that names a main service of its own owns the answer; the
	// resolver must not claim it is absent.
	t.Run("service the resolver does not know", func(t *testing.T) {
		if got := resolveMainServiceEnabled(map[string]string{}, "storefront"); got != "true" {
			t.Errorf("unknown service -> %q, want \"true\"", got)
		}
	})
}
