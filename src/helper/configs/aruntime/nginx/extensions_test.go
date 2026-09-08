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

// The worker and hash numbers are settings now, and this is the half that
// matters: the defaults are already pinned by the golden fixtures, so what needs
// saying is that a configured value actually reaches the file.
//
// bucket_size is the reason the feature exists. nginx refuses to start when a
// hostname does not fit it — on the shared proxy that is every project on the
// machine — and raising it used to mean editing Go and rebuilding the binary.
func TestProxyWorkerSettingsReachTheFile(t *testing.T) {
	preamble := proxyPreamble(map[string]string{
		"proxy/worker/processes":              "8",
		"proxy/worker/rlimit_nofile":          "65535",
		"proxy/worker/connections":            "16384",
		"proxy/server_names_hash/bucket_size": "256",
		"proxy/server_names_hash/max_size":    "4096",
	})

	for _, want := range []string{
		"worker_processes 8;",
		"worker_rlimit_nofile 65535;",
		"worker_connections 16384;",
		"server_names_hash_bucket_size  256;",
		"server_names_hash_max_size 4096;",
	} {
		if !strings.Contains(preamble, want) {
			t.Errorf("missing %q:\n%s", want, preamble)
		}
	}
}

// A key present with no value is what a config looks like mid-edit, and an empty
// directive is a file nginx will not load — on the shared proxy, for everybody.
// So empty falls back to the compiled default rather than through.
func TestAnEmptySettingFallsBackToTheDefault(t *testing.T) {
	preamble := proxyPreamble(map[string]string{
		"proxy/worker/processes": "",
	})

	if !strings.Contains(preamble, "worker_processes 2;") {
		t.Errorf("an empty setting did not fall back to the default:\n%s", preamble)
	}
}
