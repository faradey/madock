package nginx

import (
	"strings"
	"testing"
)

// The seam's whole contract is where the text lands, so this is the test that
// has to exist here rather than in the edition that registers an extension.
//
// The realip module — the one extension there is today — runs at nginx's
// post-read phase, before limit_req and limit_conn are evaluated. That is a
// property of nginx, not of file order, so the ordering pinned here is about
// something else: the directives are written above the zones so that the reason
// they exist stands next to what depends on them, and an edit that moves them
// has to say why. An extension appended after the zones would still work and
// would still be wrong to read.
//
// Proved by moving the call below the zones: this test goes red and nothing
// else in the suite notices, which is exactly the blindness it is here for.
func TestPreambleExtensionsComeBeforeTheLimitZones(t *testing.T) {
	previous := preambleExtensions
	t.Cleanup(func() { preambleExtensions = previous })

	preambleExtensions = nil
	RegisterPreambleExtension(func(map[string]string) string {
		return "# extension marker\n"
	})

	preamble := proxyPreamble(map[string]string{
		"proxy/rate_limit/enabled": "true",
		"proxy/rate_limit/rate":    "50",
		"proxy/conn_limit/enabled": "true",
		"proxy/conn_limit/per_ip":  "100",
	})

	marker := strings.Index(preamble, "# extension marker")
	rate := strings.Index(preamble, "limit_req_zone")
	conn := strings.Index(preamble, "limit_conn_zone")

	if marker == -1 || rate == -1 || conn == -1 {
		t.Fatalf("expected the marker and both zones:\n%s", preamble)
	}
	if marker > rate || marker > conn {
		t.Errorf("an extension must be rendered above both zones, got marker=%d rate=%d conn=%d:\n%s",
			marker, rate, conn, preamble)
	}
}

// Community registers nothing, and the generated file has to show it: an empty
// extension list must add no whitespace, no comment, nothing. The golden
// fixtures would catch a stray newline eventually; this says it directly.
func TestNoExtensionsRenderNothing(t *testing.T) {
	previous := preambleExtensions
	t.Cleanup(func() { preambleExtensions = previous })

	preambleExtensions = nil

	if got := PreambleExtensions(map[string]string{}); got != "" {
		t.Errorf("with no extensions registered the seam rendered %q", got)
	}
}
