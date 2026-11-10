package configs

import (
	"strings"
	"testing"
)

// The rule is Compose's, measured on Compose v5.7.0 with `docker compose
// config` on a file whose top-level name was each of these.
func TestComposeKeyIsComposesOwnRule(t *testing.T) {
	for name, want := range map[string]string{
		"madock_ShopU":    "madock_shopu",
		"madock_shop+1":   "madock_shop1",
		"madock_Shop-U_x": "madock_shop-u_x",
		"Shop:V2":         "shopv2",
		"shop":            "shop",
	} {
		if got := ComposeKey(name); got != want {
			t.Errorf("ComposeKey(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestNormalizeComposeNameTrimsLeadingSeparators(t *testing.T) {
	if got := NormalizeComposeName("_-Shop"); got != "shop" {
		t.Errorf("NormalizeComposeName = %q, want shop", got)
	}
}

// Two names Compose runs as one project: the second is refused, and the
// refusal names the first. Setting the first up again is not a clash.
func TestComposeNameTakenByFindsTheProjectComposeWouldShare(t *testing.T) {
	entries := []ProjectEntry{{Name: "ShopU"}, {Name: "shop1"}, {Name: "other"}}

	for name, want := range map[string]string{
		"shopu":  "ShopU",
		"SHOPU":  "ShopU",
		"shop+1": "shop1",
		"ShopU":  "",
		"shop1":  "",
		"shop2":  "",
		"shop_u": "",
	} {
		if got := composeNameTakenBy(name, entries, noScopes); got != want {
			t.Errorf("composeNameTakenBy(%q) = %q, want %q", name, got, want)
		}
	}
}

func noScopes(string) []string { return nil }

// Project shop in scope staging runs as madock_shop-staging. A project called
// shop-staging took over its containers and its database volume on a VM with
// 4.5.57; the setup is refused now, and so is the scope in the other order.
func TestAScopeHoldsItsComposeNameToo(t *testing.T) {
	entries := []ProjectEntry{{Name: "shop"}, {Name: "blog-qa"}}
	scopes := func(name string) []string {
		if name == "shop" {
			return []string{"Staging"}
		}
		return nil
	}

	if got := composeNameTakenBy("shop-staging", entries, scopes); got != "shop in scope Staging" {
		t.Errorf("composeNameTakenBy(shop-staging) = %q, want shop in scope Staging", got)
	}
	if got := composeNameTakenBy("shop-live", entries, scopes); got != "" {
		t.Errorf("composeNameTakenBy(shop-live) = %q, want none", got)
	}

	if got := scopeComposeClash("blog", "qa", entries, scopes); !strings.Contains(got, "blog-qa") || !strings.Contains(got, "madock_blog-qa") {
		t.Errorf("scope qa of blog: %q, want a refusal naming blog-qa", got)
	}
	if got := scopeComposeClash("shop", "staging", entries, scopes); got != "" {
		t.Errorf("a project's own scope is no clash with itself: %q", got)
	}
	if got := scopeComposeClash("blog", "live", entries, scopes); got != "" {
		t.Errorf("scope live of blog: %q, want none", got)
	}
}

// Scopes are read off the registry config's keys; default is the project's own
// name and no extra one.
func TestScopesInSkipsDefault(t *testing.T) {
	got := scopesIn(map[string]string{
		"scopes/default/path":   "/x",
		"scopes/staging/path":   "/x",
		"scopes/staging/db/x":   "1",
		"scopes/qa/php/version": "8.3",
		"activeScope":           "staging",
	})
	if strings.Join(got, ",") != "qa,staging" {
		t.Errorf("scopesIn = %v, want qa, staging", got)
	}
}
