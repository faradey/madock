package install

import (
	"os"
	"strings"
	"testing"
)

// Every address of the project's Redis that install writes goes through
// projectRedisURL; one spelled out in install.go would skip the rewriters and
// leave a platform without the password the server asks for.
func TestInstallSpellsOutNoRedisAddress(t *testing.T) {
	src, err := os.ReadFile("install.go")
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(src), "\n") {
		if strings.Contains(line, "redis://redisdb") {
			t.Errorf("install.go:%d spells out the Redis address; use projectRedisURL: %s", i+1, strings.TrimSpace(line))
		}
	}
}

// Without a rewriter — this edition — the address is exactly what Medusa,
// Saleor and Spree have always found in their .env.
func TestRedisURLIsUnchangedWithoutARewriter(t *testing.T) {
	saved := redisURLRewriters
	redisURLRewriters = nil
	t.Cleanup(func() { redisURLRewriters = saved })

	cases := map[string]string{
		"":  "redis://redisdb:6379",
		"0": "redis://redisdb:6379/0",
		"1": "redis://redisdb:6379/1",
	}
	for db, want := range cases {
		if got := projectRedisURL(map[string]string{}, db); got != want {
			t.Errorf("db %q: got %q, want %q", db, got, want)
		}
	}
}

// A registered rewriter sees the project's configuration and the address
// madock built; rewriters run in the order they were registered.
func TestRegisteredRewritersShapeTheAddressInOrder(t *testing.T) {
	saved := redisURLRewriters
	redisURLRewriters = nil
	t.Cleanup(func() { redisURLRewriters = saved })

	RegisterRedisURLRewriter(func(conf map[string]string, address string) string {
		return address + "?a=" + conf["marker"]
	})
	RegisterRedisURLRewriter(func(_ map[string]string, address string) string {
		return address + "&b"
	})

	got := projectRedisURL(map[string]string{"marker": "x"}, "0")
	if want := "redis://redisdb:6379/0?a=x&b"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
