package configs

import "testing"

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
