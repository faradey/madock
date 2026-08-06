package configs

import "testing"

func TestResolveMainService(t *testing.T) {
	cases := []struct {
		language string
		fallback string
		want     string
	}{
		{"nodejs", "php", "nodejs"},
		{"python", "php", "python"},
		{"golang", "php", "golang"},
		{"ruby", "php", "ruby"},
		{"none", "php", "app"},
		{"php", "php", "php"},
		{"", "php", "php"},
		// A caller that knows its platform's container passes it as the
		// fallback, and a php project keeps it.
		{"php", "app", "app"},
		// An unrecognised language is not guessed at.
		{"elixir", "php", "php"},
	}

	for _, c := range cases {
		conf := map[string]string{}
		if c.language != "" {
			conf["language"] = c.language
		}
		if got := ResolveMainService(conf, c.fallback); got != c.want {
			t.Errorf("language %q, fallback %q → %q, want %q", c.language, c.fallback, got, c.want)
		}
	}
}
