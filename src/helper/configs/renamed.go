package configs

import "github.com/faradey/madock/v4/src/helper/deprecation"

// renamedKeys maps a key's old name to the one it moved to. The pairs live in
// the deprecation ledger, with the major release that stops reading them.
//
// A migration moves the key in every file madock can reach once, and that is
// not every file that will ever hold it: a project's .madock/config.xml is
// committed to its repository, so a deploy brings the old spelling back into
// every new release directory, and a copy written from an older README carries
// it too. Read as unknown, the old name would be dropped without a word and the
// setting would fall back to the default. So each layer is read through this
// map before the layers merge: the old name still works, and a layer that holds
// both keeps the new one — that is the deliberate one.
var renamedKeys = deprecation.RenamedKeys()

// templateNames is every renamed key, template-only ones included: what a
// copied template may still ask for, and what config:set redirects.
var templateNames = deprecation.TemplateNames()

// RenamedTo reports the current name of a renamed key.
func RenamedTo(name string) (string, bool) {
	newName, ok := templateNames[name]
	return newName, ok
}

// MirrorRenamed gives every old name the current name's value, for templates.
//
// A template a project copied into its .madock/docker/ still says the old name,
// and that copy is committed: every deploy brings it back, so no migration can
// fix it for good. Answering the old name keeps the copy rendering what it
// rendered. Only template values get this — config:list and config:set see the
// current names alone.
func MirrorRenamed(values map[string]string) {
	for oldName, newName := range templateNames {
		if value, ok := values[newName]; ok {
			values[oldName] = value
		}
	}
}

// applyRenamed rewrites one layer's old names to the current ones, in place.
func applyRenamed(layer map[string]string) {
	if layer == nil {
		return
	}
	for oldName, newName := range renamedKeys {
		value, ok := layer[oldName]
		if !ok {
			continue
		}
		if _, taken := layer[newName]; !taken {
			layer[newName] = value
		}
		delete(layer, oldName)
	}
}
