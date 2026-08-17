package nginx

import (
	"strings"
	"testing"
)

// The rate limit was written to catch a request loop, not an attacker, and its
// default said so: 1000 requests a second from one address is a permission. The
// zone is unchanged; what it allows is not.
func TestProxyPreamble_RateZoneFollowsTheSetting(t *testing.T) {
	preamble := proxyPreamble(map[string]string{
		"proxy/rate_limit/enabled": "true",
		"proxy/rate_limit/rate":    "50",
	})

	if !strings.Contains(preamble, "limit_req_zone $binary_remote_addr zone=general:10m rate=50r/s;") {
		t.Fatalf("the rate zone is missing or wrong:\n%s", preamble)
	}
}

// The half nothing answered: a request that never finishes spends no rate at
// all, so slow connections hold every worker while staying under any
// per-second limit.
func TestProxyPreamble_ConnectionZoneIsDeclared(t *testing.T) {
	preamble := proxyPreamble(map[string]string{
		"proxy/conn_limit/enabled": "true",
	})

	if !strings.Contains(preamble, "limit_conn_zone $binary_remote_addr zone=addr:10m;") {
		t.Fatalf("the connection zone is missing:\n%s", preamble)
	}
}

// Both are switches, and a zone declared without a matching directive — or a
// directive without its zone — is a configuration nginx refuses to start on.
func TestProxyPreamble_NeitherZoneWhenTurnedOff(t *testing.T) {
	preamble := proxyPreamble(map[string]string{
		"proxy/rate_limit/enabled": "false",
		"proxy/conn_limit/enabled": "false",
	})

	for _, directive := range []string{"limit_req_zone", "limit_conn_zone"} {
		if strings.Contains(preamble, directive) {
			t.Errorf("%s is declared even though it is turned off:\n%s", directive, preamble)
		}
	}
}
