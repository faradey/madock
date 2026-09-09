package configs

import (
	"strings"
	"testing"
)

// A community build must behave exactly as it did before the seam existed, and
// "exactly" is the whole claim: a seam that quietly changed the merge would be
// the feature it is meant not to be.
func TestWithoutAnExtensionTheLayersMergeUnchanged(t *testing.T) {
	RegisterLayerExtension(nil)

	project := map[string]string{"php/version": "8.3", "nginx/hosts/www/name": "www.test"}
	machine := map[string]string{"php/version": "8.4"}
	general := map[string]string{"timezone": "UTC"}

	adjustLayers(project, machine, general)

	if project["php/version"] != "8.3" || len(project) != 2 {
		t.Errorf("the project layer was touched: %v", project)
	}
	if machine["php/version"] != "8.4" || len(machine) != 1 {
		t.Errorf("the machine layer was touched: %v", machine)
	}
}

// The flag has to say what it cannot do. A command that accepts `--machine`,
// prints nothing and changes nothing is worse than one that refuses.
func TestTheFlagSaysWhichEditionDoesIt(t *testing.T) {
	RegisterLayerExtension(nil)

	err := DeclareLayerKeyRemoved("someproject", "nginx/hosts/www", "default")
	if err == nil {
		t.Fatal("a community build accepted a declaration it cannot honour")
	}
	if !strings.Contains(err.Error(), "madock-pro") {
		t.Errorf("the refusal does not name the edition: %v", err)
	}
	if keys := LayerKeysRemoved("someproject"); keys != nil {
		t.Errorf("a community build reported keys kept out: %v", keys)
	}
}

// stubExtension is the smallest thing that proves the seam is wired: it removes
// one key from the project layer.
type stubExtension struct{ removed string }

func (s *stubExtension) Adjust(project, machine, general map[string]string) {
	delete(project, s.removed)
}
func (s *stubExtension) Declare(string, string, string) error { return nil }
func (s *stubExtension) Declared(string) []string             { return []string{s.removed} }

// The seam is called at the moment the layers are still separate, and what an
// extension removes from the project layer does not come back from the merge.
func TestAnExtensionCanRemoveFromTheProjectLayer(t *testing.T) {
	RegisterLayerExtension(&stubExtension{removed: "nginx/hosts/www/name"})
	t.Cleanup(func() { RegisterLayerExtension(nil) })

	project := map[string]string{"nginx/hosts/www/name": "www.test", "php/version": "8.3"}
	machine := map[string]string{}
	general := map[string]string{}

	adjustLayers(project, machine, general)

	if value, still := project["nginx/hosts/www/name"]; still {
		t.Errorf("the extension's removal did not reach the layer: %q", value)
	}
	if project["php/version"] != "8.3" {
		t.Errorf("it took something else with it: %v", project)
	}
}

// The general config is cached for the process and shared by every project read
// in it — the shared proxy generator reads several — so an extension must not be
// able to remove a key from it for one project and have it stay gone for the
// next.
func TestTheGeneralLayerHandedOverIsACopy(t *testing.T) {
	RegisterLayerExtension(&generalEater{})
	t.Cleanup(func() { RegisterLayerExtension(nil) })

	general := map[string]string{"timezone": "UTC"}
	adjustLayers(map[string]string{}, map[string]string{}, general)

	if general["timezone"] != "UTC" {
		t.Error("an extension emptied the process-wide general config; the next project would inherit that")
	}
}

type generalEater struct{}

func (generalEater) Adjust(project, machine, general map[string]string) {
	for key := range general {
		delete(general, key)
	}
}
func (generalEater) Declare(string, string, string) error { return nil }
func (generalEater) Declared(string) []string             { return nil }
