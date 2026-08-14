package nginx

import (
	"strings"
	"testing"
)

// TestProxyPreamble covers the directives that exist once for the whole proxy.
//
// Two of them are load-bearing in ways nothing else checks: the WebSocket upgrade
// map, which every `location /` in the generated configuration refers to — a
// missing map is an unknown variable and nginx does not start at all — and the
// absence of worker_priority, which was set for years and never once applied,
// because lowering a nice value needs CAP_SYS_NICE and a container does not have
// it. Each start logged two alerts about it in the log where real faults appear.
func TestProxyPreamble(t *testing.T) {
	t.Run("always present", func(t *testing.T) {
		preamble := proxyPreamble(map[string]string{})

		for _, want := range []string{
			"worker_processes 2;",
			"worker_rlimit_nofile 200000;",
			"map $http_upgrade $connection_upgrade {",
			"access_log /var/log/nginx/access.log main;",
		} {
			if !strings.Contains(preamble, want) {
				t.Errorf("preamble is missing %q:\n%s", want, preamble)
			}
		}
	})

	t.Run("no worker_priority", func(t *testing.T) {
		if strings.Contains(proxyPreamble(map[string]string{}), "worker_priority") {
			t.Error("worker_priority is back: it cannot work without CAP_SYS_NICE and only " +
				"produces two [alert] lines per start")
		}
	})

	t.Run("rate limiting off by default", func(t *testing.T) {
		if strings.Contains(proxyPreamble(map[string]string{}), "limit_req_zone") {
			t.Error("a rate limiting zone was declared without being asked for")
		}
	})

	t.Run("rate limiting carries the configured rate", func(t *testing.T) {
		preamble := proxyPreamble(map[string]string{
			"proxy/rate_limit/enabled": "true",
			"proxy/rate_limit/rate":    "40",
		})
		if !strings.Contains(preamble, "zone=general:10m rate=40r/s;") {
			t.Errorf("the configured rate is not in the zone:\n%s", preamble)
		}
	})

	t.Run("gzip only when enabled", func(t *testing.T) {
		if strings.Contains(proxyPreamble(map[string]string{}), "gzip on;") {
			t.Error("gzip was enabled without being asked for")
		}
		if !strings.Contains(proxyPreamble(map[string]string{"proxy/gzip/enabled": "true"}), "gzip on;") {
			t.Error("gzip was asked for and is absent")
		}
	})
}
