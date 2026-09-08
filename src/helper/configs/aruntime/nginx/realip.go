package nginx

import (
	"net"
	"strings"
)

// Trusted upstream addresses for the realip module.
//
// Why this exists at all: every per-address protection the shared proxy has is
// keyed on `$binary_remote_addr` — the rate limit zone and the connection limit
// zone both — and so is the access log. Put a CDN in front of the proxy and that
// address stops being the client: it becomes the CDN's edge, of which there are a
// few dozen. The limits then count every visitor in the world as the same handful
// of addresses, which is worse than having no limit at all: one busy shop can
// exhaust the burst for everybody, and a single attacker is indistinguishable
// from the crowd behind the same edge. The log goes the same way — the field that
// answers "who did this" starts answering "Cloudflare".
//
// The realip module rewrites `$remote_addr` from a header, but only for requests
// that arrive from an address on this list. That restriction is the whole
// security of it: without it any client could send `CF-Connecting-IP` and choose
// the address the rate limiter counts, and the admin allow-lists trust.
//
// Ranges are shipped rather than fetched. A proxy that phones a vendor at start
// is a proxy that does not start when the vendor is down, and these change on the
// order of once a year. The cost is that they go stale silently, which is why the
// date and the source are recorded here and why `trusted` also accepts explicit
// CIDRs for anything else in front of the proxy.
//
// Fetched 2026-09-08 from https://www.cloudflare.com/ips-v4 and
// https://www.cloudflare.com/ips-v6.
var cloudflareRanges = []string{
	"173.245.48.0/20",
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"141.101.64.0/18",
	"108.162.192.0/18",
	"190.93.240.0/20",
	"188.114.96.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",
	"162.158.0.0/15",
	"104.16.0.0/13",
	"104.24.0.0/14",
	"172.64.0.0/13",
	"131.0.72.0/22",
	"2400:cb00::/32",
	"2606:4700::/32",
	"2803:f800::/32",
	"2405:b500::/32",
	"2405:8100::/32",
	"2a06:98c0::/29",
	"2c0f:f248::/32",
}

// TrustedRealIPRanges turns the `proxy/real_ip/trusted` value into the list of
// addresses nginx will accept a client address from.
//
// The value is a comma-separated list where the word `cloudflare` expands to the
// shipped ranges and everything else is taken as a literal address or CIDR, so a
// stack behind Cloudflare *and* a company load balancer can name both.
//
// An entry that is not a valid address or CIDR is dropped rather than passed
// through. nginx would refuse to start on it, taking every project on the machine
// down at once for a typo in one project's config — and a proxy that will not
// start is the most expensive failure this file can cause. The second return
// value names what was dropped so the caller can say so in the generated file
// instead of leaving it invisible.
func TrustedRealIPRanges(trusted string) (ranges []string, rejected []string) {
	seen := make(map[string]bool)

	add := func(candidate string) {
		if candidate == "" || seen[candidate] {
			return
		}
		seen[candidate] = true
		ranges = append(ranges, candidate)
	}

	for _, part := range strings.Split(trusted, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.EqualFold(part, "cloudflare") {
			for _, cidr := range cloudflareRanges {
				add(cidr)
			}
			continue
		}

		if _, _, err := net.ParseCIDR(part); err == nil {
			add(part)
			continue
		}

		// A bare address is legal in set_real_ip_from, and is how a single
		// upstream load balancer is named.
		if net.ParseIP(part) != nil {
			add(part)
			continue
		}

		rejected = append(rejected, part)
	}

	return ranges, rejected
}
