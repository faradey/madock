package configs

import "testing"

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
		if got := composeNameTakenBy(name, entries); got != want {
			t.Errorf("composeNameTakenBy(%q) = %q, want %q", name, got, want)
		}
	}
}
