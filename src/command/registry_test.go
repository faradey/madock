package command

import "testing"

// TestIsGlobal covers the two ways a command can be excused from needing a project,
// and the order between them.
//
// The flag is madock's own: unmarked means project-scoped, so a check forgotten
// inside a project command cannot be forgotten silently. The resolver is for a layer
// built on top — madock-pro registers around a hundred and ten aliases madock knows
// nothing about, and whole families of them act on the machine. Without a way to say
// so, its `firewall:status` was refused outside a project, which is the only place
// anybody runs it.
func TestIsGlobal(t *testing.T) {
	t.Cleanup(func() { scopeResolvers = nil })

	project := &Definition{Aliases: []string{"start"}}
	global := &Definition{Aliases: []string{"help"}, Global: true}
	layer := &Definition{Aliases: []string{"firewall:status"}}

	if IsGlobal(project) {
		t.Error("an unmarked command must be project-scoped")
	}
	if !IsGlobal(global) {
		t.Error("the flag must be honoured")
	}
	if IsGlobal(layer) {
		t.Error("a command no resolver has claimed is project-scoped")
	}

	AddScopeResolver(func(def *Definition) (bool, bool) {
		for _, alias := range def.Aliases {
			if alias == "firewall:status" {
				return true, true
			}
		}
		return false, false
	})

	if !IsGlobal(layer) {
		t.Error("a resolver said this one is global and was not heard")
	}
	if IsGlobal(project) {
		t.Error("an undecided resolver must leave the flag in charge")
	}

	// A resolver may also pin a command as project-scoped, which has to win over a
	// flag set further down.
	AddScopeResolver(func(def *Definition) (bool, bool) {
		for _, alias := range def.Aliases {
			if alias == "help" {
				return false, true
			}
		}
		return false, false
	})

	if IsGlobal(global) {
		t.Error("a resolver's explicit answer must beat the flag")
	}
}
