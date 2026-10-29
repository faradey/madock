package clone

import "testing"

// A failed clone removes projects/<name>/. Every name accepted here is one that
// removal may be pointed at, so the refusals are the guard against deleting
// projects/ itself or a directory outside it.
func TestNameRefusal(t *testing.T) {
	refused := []string{"", "  ", "a/b", "../x", `a\b`, "..", "a.b", "a b"}
	for _, name := range refused {
		if nameRefusal(name) == "" {
			t.Errorf("%q was accepted", name)
		}
	}
	for _, name := range []string{"dbg1", "shop-clean-it4", "shop_update"} {
		if refusal := nameRefusal(name); refusal != "" {
			t.Errorf("%q was refused: %s", name, refusal)
		}
	}
}
