package docker

import (
	"errors"
	"strings"
	"testing"
)

// noContainers is a docker where nothing carries any label.
func noContainers(string) ([]string, error) { return nil, nil }

// The refusals are what stands between a name typed on a server and
// `docker volume rm` of a live database, so each is tested on its own.
func TestOrphanRemovalRefusesARegisteredProject(t *testing.T) {
	known := map[string]bool{"ShopU": true}

	refusal := orphanRemovalRefusal("shopu", known, noScopes, noContainers)
	if !strings.Contains(refusal, "ShopU") {
		t.Errorf("the Compose name of registered ShopU was not refused, or not named: %q", refusal)
	}
}

func TestOrphanRemovalRefusesARegisteredProjectsScope(t *testing.T) {
	known := map[string]bool{"Shop": true}
	scopes := func(name string) ([]string, bool) {
		if name == "Shop" {
			return []string{"Staging"}, true
		}
		return nil, true
	}

	refusal := orphanRemovalRefusal("shop-staging", known, scopes, noContainers)
	if !strings.Contains(refusal, "Shop in scope Staging") {
		t.Errorf("shop-staging, Shop's scope Staging, was not refused as that: %q", refusal)
	}
}

// A project whose registry config does not parse has scopes nobody can name;
// its <key>-<anything> may be one of them.
func TestOrphanRemovalRefusesAScopeOfAnUnreadableProject(t *testing.T) {
	known := map[string]bool{"shop": true}
	scopes := func(string) ([]string, bool) { return nil, false }

	if refusal := orphanRemovalRefusal("shop-staging", known, scopes, noContainers); refusal == "" {
		t.Error("shop-staging was allowed while shop's scopes cannot be read")
	}
	if refusal := orphanRemovalRefusal("shopx", known, scopes, noContainers); refusal != "" {
		t.Errorf("shopx is not shop's and was refused: %q", refusal)
	}
}

func TestOrphanRemovalRefusesWhileAContainerCarriesTheLabel(t *testing.T) {
	var asked string
	containers := func(label string) ([]string, error) {
		asked = label
		return []string{"madock_gone-php-1"}, nil
	}

	refusal := orphanRemovalRefusal("gone", map[string]bool{}, noScopes, containers)
	if !strings.Contains(refusal, "madock_gone-php-1") {
		t.Errorf("a container under madock_gone did not stop the removal: %q", refusal)
	}
	if asked != "madock_gone" {
		t.Errorf("asked docker for containers of %q, want madock_gone", asked)
	}
}

// Not knowing is not "none": a volume is not deleted on a guess.
func TestOrphanRemovalRefusesWhenDockerCannotBeAsked(t *testing.T) {
	containers := func(string) ([]string, error) { return nil, errors.New("Cannot connect to the Docker daemon") }

	if refusal := orphanRemovalRefusal("gone", map[string]bool{}, noScopes, containers); refusal == "" {
		t.Error("the removal was allowed with docker unreachable")
	}
}

// Compose would turn `Shop,V2` into shopv2, a project nobody typed.
func TestOrphanRemovalRefusesANameNotAsPrinted(t *testing.T) {
	for _, name := range []string{"", "Gone", "go,ne", "gone project"} {
		if refusal := orphanRemovalRefusal(name, map[string]bool{}, noScopes, noContainers); refusal == "" {
			t.Errorf("%q was allowed", name)
		}
	}
}

func TestOrphanRemovalAllowsAForgottenProject(t *testing.T) {
	known := map[string]bool{"live": true}

	if refusal := orphanRemovalRefusal("gone", known, noScopes, noContainers); refusal != "" {
		t.Errorf("a forgotten project with nothing running was refused: %q", refusal)
	}
}

// The plan keeps exactly one project's leftovers, and goes by the rule the
// list goes by: with the refusals out of the way, a registered name still
// yields nothing to remove.
func TestOrphanRemovalPlanTakesOneProjectOnly(t *testing.T) {
	known := map[string]bool{"live": true}
	volumes := []string{
		"madock_gone_dbdata\tmadock_gone",
		"madock_gone-x_dbdata\tmadock_gone-x",
		"madock_live_dbdata\tmadock_live",
		"stranger_data\tstranger",
	}
	networks := []string{"madock_gone_default\tmadock_gone"}
	images := []string{"aaa\tmadock_gone", "bbb\tmadock_live"}
	ids := map[string]PruneImage{"aaa": {ID: "aaa", Name: "madock_gone-php"}, "bbb": {ID: "bbb", Name: "madock_live-php"}}

	plan := orphanRemovalPlan("gone", volumes, networks, images, ids, known, noScopes)
	if len(plan.Volumes) != 1 || plan.Volumes[0] != "madock_gone_dbdata" {
		t.Errorf("volumes = %v, want only madock_gone_dbdata", plan.Volumes)
	}
	if len(plan.Networks) != 1 || plan.Networks[0] != "madock_gone_default" {
		t.Errorf("networks = %v", plan.Networks)
	}
	if len(plan.Images) != 1 || plan.Images[0].ID != "aaa" {
		t.Errorf("images = %+v, want only aaa", plan.Images)
	}

	if live := orphanRemovalPlan("live", volumes, networks, images, ids, known, noScopes); !live.Empty() {
		t.Errorf("a registered project's resources were planned for removal: %+v", live)
	}
}
