package buildcache

import (
	"strings"
	"testing"
)

const compose = `services:
  php:
    init: true
    build:
      context: ctx
      dockerfile: php.Dockerfile
    restart: no
  db:
    build:
      context: ctx
      dockerfile: db.Dockerfile
  mailpit:
    image: axllent/mailpit:v1.31.0
`

func reader(files map[string]string) func(string, string) string {
	return func(context, dockerfile string) string { return files[context+"/"+dockerfile] }
}

// Each built service gets a cache scoped by its Dockerfile; a pulled image
// gets nothing, and neither does a service whose Dockerfile cannot be read.
func TestApplyScopesEachBuildByItsDockerfile(t *testing.T) {
	out := Apply(compose, reader(map[string]string{"ctx/php.Dockerfile": "FROM ubuntu\nRUN php"}))

	if strings.Count(out, "cache_from:") != 1 || strings.Count(out, "cache_to:") != 1 {
		t.Fatalf("want one cache, on php only:\n%s", out)
	}
	php := strings.Split(out, "  db:")[0]
	if !strings.Contains(php, "type=gha,scope=madock-php-") || !strings.Contains(php, ",mode=min") {
		t.Errorf("php's cache is not on GitHub's cache with its own scope:\n%s", php)
	}
	if !strings.Contains(out, "    restart: no") || !strings.Contains(out, "    image: axllent/mailpit:v1.31.0") {
		t.Errorf("the rest of the file moved:\n%s", out)
	}
}

// Two projects building the same Dockerfile share a scope; a different
// Dockerfile never does. That is the whole point: the shards build the same
// images for different test projects.
func TestScopeFollowsTheContent(t *testing.T) {
	block := "      context: ctx\n      dockerfile: php.Dockerfile"
	a := Scope("php.Dockerfile", "FROM ubuntu\nRUN php8.3", block)
	b := Scope("php.Dockerfile", "FROM ubuntu\nRUN php8.3", block)
	c := Scope("php.Dockerfile", "FROM ubuntu\nRUN php8.4", block)
	if a != b {
		t.Errorf("the same build got two scopes: %s, %s", a, b)
	}
	if a == c {
		t.Errorf("two different Dockerfiles share scope %s", a)
	}
}

// Applied twice, nothing doubles: a rebuild renders the file again, and a
// second pass over a file that already has its cache leaves it.
func TestApplyIsIdempotent(t *testing.T) {
	files := reader(map[string]string{"ctx/php.Dockerfile": "x", "ctx/db.Dockerfile": "y"})
	once := Apply(compose, files)
	if twice := Apply(once, files); twice != once {
		t.Errorf("a second pass changed the file:\n%s", twice)
	}
}

// Off unless the variable says gha.
func TestEnabledOnlyForGha(t *testing.T) {
	for value, want := range map[string]bool{"": false, "gha": true, "local": false} {
		t.Setenv(Env, value)
		if Enabled() != want {
			t.Errorf("%s=%q: Enabled() = %v", Env, value, !want)
		}
	}
}
