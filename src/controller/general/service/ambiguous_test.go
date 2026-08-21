package service

import (
	"sort"
	"strings"
	"testing"
)

// Two platforms claim `storefront` (medusa, spree) and two claim `messenger`
// (shopware, sylius). GetByShort used to range over the map and take the first
// match, and Go randomises map iteration — so the answer was not merely
// ambiguous, it was random. Measured on 2026-08-21 against the old code: 200
// calls returned medusa/storefront 195 times and spree/storefront 5.
//
// `madock service:enable storefront` in a Spree project therefore set medusa's
// key almost always and its own occasionally, and neither outcome said anything.
//
// Repetition is the assertion here. One call proves nothing about a bug whose
// whole shape is that it usually goes one way.
func TestGetByShortIsDeterministic(t *testing.T) {
	for _, short := range []string{"storefront", "messenger", "xdebug", "db"} {
		spree := func() string { return "spree" }
		first, _ := resolveShort(short, spree)
		for i := 0; i < 200; i++ {
			if got, _ := resolveShort(short, spree); got != first {
				t.Fatalf("resolveShort(%q) answered %q and then %q", short, first, got)
			}
		}
	}
}

// A name nobody claims comes back as itself: long keys are handed to this
// function too, and turning one into something else would be worse than not
// resolving it.
func TestGetByShortLeavesUnknownNamesAlone(t *testing.T) {
	for _, name := range []string{"medusa/storefront", "nothing-like-this", "php/xdebug"} {
		got, _ := resolveShort(name, func() string { return "" })
		if got != name {
			t.Errorf("resolveShort(%q) = %q, want it unchanged", name, got)
		}
	}
}

// The platform decides, and it is the only thing that can: both keys are real
// and both are somebody's `storefront`.
func TestThePlatformPicksBetweenClaimants(t *testing.T) {
	cases := []struct {
		short    string
		platform string
		want     string
	}{
		{"storefront", "spree", "spree/storefront"},
		{"storefront", "medusa", "medusa/storefront"},
		{"messenger", "shopware", "shopware/messenger"},
		{"messenger", "sylius", "sylius/messenger"},
	}

	for _, c := range cases {
		got, candidates := resolveShort(c.short, func() string { return c.platform })
		if candidates != nil {
			t.Errorf("%s on %s was refused, and it should not have been: %v", c.short, c.platform, candidates)
			continue
		}
		if got != c.want {
			t.Errorf("%s on %s resolved to %q, want %q", c.short, c.platform, got, c.want)
		}
	}
}

// A contested name in a project belonging to neither claimant is refused, with
// both candidates named. Guessing here is what the old code did, and the guess
// was silent — which is the worse half: enabling the wrong platform's key
// changes the config, reports success and does nothing to the stack.
func TestAContestedNameIsRefusedRatherThanGuessed(t *testing.T) {
	for _, platform := range []string{"", "magento2", "shopify"} {
		got, candidates := resolveShort("storefront", func() string { return platform })

		if candidates == nil {
			t.Errorf("storefront on %q resolved to %q instead of being refused", platform, got)
			continue
		}
		if len(candidates) != 2 {
			t.Errorf("the refusal named %v, and the reader needs both claimants", candidates)
		}
	}
}

// The guard against this class coming back. A short name claimed by two
// platforms is fine — the project's platform decides — but a short name claimed
// twice *within one platform*, or by an entry with no platform prefix, cannot be
// resolved by anything and would put the random answer straight back.
func TestEveryAmbiguousNameIsResolvableByPlatform(t *testing.T) {
	byShort := map[string][]string{}
	for key, short := range serviceMap {
		byShort[short] = append(byShort[short], key)
	}

	for short, keys := range byShort {
		if len(keys) < 2 {
			continue
		}
		sort.Strings(keys)

		seen := map[string]bool{}
		for _, key := range keys {
			platform := strings.SplitN(key, "/", 2)[0]
			if platform == key {
				t.Errorf("%q is claimed by %v, and %q has no platform prefix — nothing can tell them apart",
					short, keys, key)
				continue
			}
			if seen[platform] {
				t.Errorf("%q is claimed twice by the same platform in %v — the platform cannot decide between them",
					short, keys)
			}
			seen[platform] = true
		}
	}
}
