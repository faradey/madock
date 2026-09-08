package nginx

import (
	"strings"
	"testing"
)

// Off by default is the security of the feature, not a preference: the realip
// module rewrites the address every per-client limit is keyed on, so a proxy
// that trusts a header on a directly reachable machine lets the client pick the
// address the rate limiter counts.
func TestRealIP_NothingWhenTurnedOff(t *testing.T) {
	preamble := proxyPreamble(map[string]string{
		"proxy/real_ip/enabled": "false",
		"proxy/real_ip/trusted": "cloudflare",
	})

	for _, directive := range []string{"set_real_ip_from", "real_ip_header", "real_ip_recursive"} {
		if strings.Contains(preamble, directive) {
			t.Errorf("%s is emitted even though real_ip is off:\n%s", directive, preamble)
		}
	}
}

// The keyword is the point: nobody should be pasting twenty-two CIDRs into a
// project config, and a hand-copied list is a list that goes stale in one place
// and not another.
func TestRealIP_CloudflareKeywordExpands(t *testing.T) {
	preamble := proxyPreamble(map[string]string{
		"proxy/real_ip/enabled": "true",
		"proxy/real_ip/trusted": "cloudflare",
		"proxy/real_ip/header":  "CF-Connecting-IP",
	})

	// one v4 range, one v6 range, and the header — enough to prove the
	// expansion happened without pinning the vendor's whole list twice
	for _, want := range []string{
		"set_real_ip_from 173.245.48.0/20;",
		"set_real_ip_from 2606:4700::/32;",
		"real_ip_header CF-Connecting-IP;",
	} {
		if !strings.Contains(preamble, want) {
			t.Errorf("missing %q:\n%s", want, preamble)
		}
	}

	if strings.Contains(preamble, "real_ip_recursive") {
		t.Errorf("recursive was not asked for but is on:\n%s", preamble)
	}
}

// The rewritten address is what the limit zones count — the realip module runs
// at post-read and limit_req at preaccess, so that holds regardless of where the
// directives sit in the file. This pins the layout rather than the behaviour: the
// block is written above the zones so the reason it exists is visible next to
// what depends on it, and a later edit that moves it should have to say why.
func TestRealIP_ComesBeforeTheLimitZones(t *testing.T) {
	preamble := proxyPreamble(map[string]string{
		"proxy/real_ip/enabled":    "true",
		"proxy/real_ip/trusted":    "cloudflare",
		"proxy/rate_limit/enabled": "true",
		"proxy/rate_limit/rate":    "50",
		"proxy/conn_limit/enabled": "true",
	})

	realIP := strings.Index(preamble, "set_real_ip_from")
	rate := strings.Index(preamble, "limit_req_zone")
	conn := strings.Index(preamble, "limit_conn_zone")

	if realIP == -1 || rate == -1 || conn == -1 {
		t.Fatalf("expected all three blocks:\n%s", preamble)
	}
	if realIP > rate || realIP > conn {
		t.Errorf("real_ip must precede both zones, got real_ip=%d rate=%d conn=%d:\n%s", realIP, rate, conn, preamble)
	}
}

// A literal address or CIDR is how anything that is not Cloudflare gets named,
// and mixing them with the keyword is the case of a stack behind both a CDN and
// a company load balancer.
func TestRealIP_ExplicitEntriesAndMixing(t *testing.T) {
	preamble := proxyPreamble(map[string]string{
		"proxy/real_ip/enabled":   "true",
		"proxy/real_ip/trusted":   "cloudflare, 10.0.0.0/8, 192.0.2.7",
		"proxy/real_ip/header":    "X-Forwarded-For",
		"proxy/real_ip/recursive": "true",
	})

	for _, want := range []string{
		"set_real_ip_from 104.16.0.0/13;",
		"set_real_ip_from 10.0.0.0/8;",
		"set_real_ip_from 192.0.2.7;",
		"real_ip_header X-Forwarded-For;",
		"real_ip_recursive on;",
	} {
		if !strings.Contains(preamble, want) {
			t.Errorf("missing %q:\n%s", want, preamble)
		}
	}
}

// nginx refuses to start on a bad set_real_ip_from, which on this machine means
// every project goes down for a typo in one project's config. So a junk entry is
// dropped — and said so in the file, because a silently discarded trusted range
// looks exactly like one that works.
func TestRealIP_JunkIsDroppedAndAnnounced(t *testing.T) {
	preamble := proxyPreamble(map[string]string{
		"proxy/real_ip/enabled": "true",
		"proxy/real_ip/trusted": "10.0.0.0/8, not-an-address, 300.1.2.3",
	})

	if strings.Contains(preamble, "not-an-address") && !strings.Contains(preamble, "# real_ip: ignored 'not-an-address'") {
		t.Errorf("junk reached a directive instead of a comment:\n%s", preamble)
	}
	for _, want := range []string{
		"# real_ip: ignored 'not-an-address' — not an address or CIDR",
		"# real_ip: ignored '300.1.2.3' — not an address or CIDR",
		"set_real_ip_from 10.0.0.0/8;",
	} {
		if !strings.Contains(preamble, want) {
			t.Errorf("missing %q:\n%s", want, preamble)
		}
	}
}

// Turned on with nothing usable trusted, the module would silently trust
// nobody — which is the safe outcome, and worth a line in the file so it does
// not read as "real_ip is working".
func TestRealIP_EnabledWithNoUsableRange(t *testing.T) {
	preamble := proxyPreamble(map[string]string{
		"proxy/real_ip/enabled": "true",
		"proxy/real_ip/trusted": "nonsense",
	})

	if !strings.Contains(preamble, "# real_ip: enabled with no usable trusted range") {
		t.Errorf("the empty case is not announced:\n%s", preamble)
	}
	if strings.Contains(preamble, "real_ip_header") {
		t.Errorf("a header is trusted with no upstream to trust it from:\n%s", preamble)
	}
}

// The default header is Cloudflare's, because that is what `trusted=cloudflare`
// pairs with, and an empty header setting would otherwise emit
// `real_ip_header ;` — a file nginx will not start on.
func TestRealIP_HeaderDefaultsRatherThanEmpty(t *testing.T) {
	preamble := proxyPreamble(map[string]string{
		"proxy/real_ip/enabled": "true",
		"proxy/real_ip/trusted": "cloudflare",
	})

	if !strings.Contains(preamble, "real_ip_header CF-Connecting-IP;") {
		t.Errorf("expected the Cloudflare header by default:\n%s", preamble)
	}
	if strings.Contains(preamble, "real_ip_header ;") {
		t.Errorf("emitted a directive with no value:\n%s", preamble)
	}
}

func TestTrustedRealIPRanges_DeduplicatesAndReportsJunk(t *testing.T) {
	ranges, rejected := TrustedRealIPRanges("cloudflare, 104.16.0.0/13, 104.16.0.0/13, bad")

	seen := 0
	for _, entry := range ranges {
		if entry == "104.16.0.0/13" {
			seen++
		}
	}
	if seen != 1 {
		t.Errorf("expected 104.16.0.0/13 once, got %d: %v", seen, ranges)
	}
	if len(rejected) != 1 || rejected[0] != "bad" {
		t.Errorf("expected only 'bad' rejected, got %v", rejected)
	}
}
