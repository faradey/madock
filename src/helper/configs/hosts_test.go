package configs

import "testing"

// A host name goes into the shared proxy's `server_name` raw. One project's
// config.xml with a value nginx cannot parse there — a space, a `;`, a newline —
// made the whole proxy.conf one the proxy refused, and every project lost
// routing. GetHosts drops such a value and keeps the rest.
func TestGetHostsDropsAHostThatWouldBreakTheProxy(t *testing.T) {
	hosts := GetHosts(map[string]string{
		"nginx/hosts/base/name": "good.test",
		"nginx/hosts/bad/name":  "bad .test; rm -rf",
	})

	if len(hosts) != 1 {
		t.Fatalf("hosts = %v, want only the valid one", hosts)
	}
	if hosts[0]["name"] != "good.test" {
		t.Errorf("kept %q, want good.test", hosts[0]["name"])
	}
}

func TestIsValidHostName(t *testing.T) {
	valid := []string{"example.com", "www.example.com", "a.b-c.example.com", "*.example.com", ".example.com", "localhost", "x.test"}
	invalid := []string{"", "bad host.test", "bad;.test", "has\n.test", "a_b.test", "-leading.test", "quote\".test", "semi;colon", "brace{.test"}
	for _, h := range valid {
		if !IsValidHostName(h) {
			t.Errorf("IsValidHostName(%q) = false, want true", h)
		}
	}
	for _, h := range invalid {
		if IsValidHostName(h) {
			t.Errorf("IsValidHostName(%q) = true, want false", h)
		}
	}
}

// The key is the website code, and that is what makes this worth a test rather
// than a comment: `nginx/hosts/<key>/name` reaches Magento as MAGE_RUN_CODE, so
// a host added under a key that names no website answers 500 to every request.
func TestHostCodeDefaultsToTheKey(t *testing.T) {
	hosts := GetHosts(map[string]string{
		"nginx/hosts/base/name": "example.com",
	})

	if len(hosts) != 1 {
		t.Fatalf("hosts = %v, want one", hosts)
	}
	if hosts[0]["name"] != "example.com" || hosts[0]["code"] != "base" {
		t.Errorf("host = %v, want name=example.com code=base", hosts[0])
	}
}

// The case this was written for: www.example.com served by the website the bare
// domain is served by. Before it, the only way to have www answer at all was to
// name the key `base`, which the bare domain already used.
func TestAHostCanNameTheWebsiteItShares(t *testing.T) {
	hosts := GetHosts(map[string]string{
		"nginx/hosts/base/name": "example.com",
		"nginx/hosts/www/name":  "www.example.com",
		"nginx/hosts/www/code":  "base",
	})

	if len(hosts) != 2 {
		t.Fatalf("hosts = %v, want two", hosts)
	}

	byName := map[string]string{}
	for _, host := range hosts {
		byName[host["name"]] = host["code"]
	}
	if byName["example.com"] != "base" {
		t.Errorf("example.com is served as %q, want base", byName["example.com"])
	}
	if byName["www.example.com"] != "base" {
		t.Errorf("www.example.com is served as %q, want base — that is the whole feature", byName["www.example.com"])
	}
}

// A code key is a property of a host, not a host. The loop used to accept any
// key under hosts/, which was harmless while `name` was the only leaf and would
// have turned every `code` into a phantom host the moment one existed: named
// after the code, coded after the host, and put into server_name.
func TestACodeKeyIsNotAHostOfItsOwn(t *testing.T) {
	hosts := GetHosts(map[string]string{
		"nginx/hosts/base/name": "example.com",
		"nginx/hosts/www/name":  "www.example.com",
		"nginx/hosts/www/code":  "base",
	})

	for _, host := range hosts {
		if host["name"] == "base" {
			t.Errorf("a code key was read as a host: %v", host)
		}
	}
}

// An empty code is not a code. Written out because the alternative — an empty
// MAGE_RUN_CODE reaching the application — is the failure this feature exists
// to remove, arriving through the feature itself.
func TestAnEmptyCodeFallsBackToTheKey(t *testing.T) {
	hosts := GetHosts(map[string]string{
		"nginx/hosts/second/name": "second.example.com",
		"nginx/hosts/second/code": "",
	})

	if len(hosts) != 1 || hosts[0]["code"] != "second" {
		t.Errorf("hosts = %v, want the key as the code", hosts)
	}
}
