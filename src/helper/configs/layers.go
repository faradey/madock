package configs

// Extension point in the layering of configuration.
//
// A project's configuration is assembled from files that fill each other's
// gaps: the built-in defaults, the installation's own config, this machine's
// config for the project, and the `.madock/config.xml` committed beside the
// source. The last of those wins every key it declares, and until this seam
// existed no layer could take a key away — only supply one that was missing.
//
// That is a limitation with a shape: it is felt where the committed file
// **arrives by a deploy**, on a machine the project was not written for. A demo
// server inherits the production hostnames because the repository says so, and
// the ways out are editing a file that belongs to the repository or forking it.
// Deploying is madock-pro's, so the answer is too — this repository declares
// where a layer may be adjusted, and nothing about what the adjustment is.
//
// Same arrangement as nginx.RegisterPreambleExtension and SetSecretsProvider.
// Nothing here registers anything, so in community the layers merge exactly as
// they did before the seam existed.

// LayerExtension adjusts the layers before they are merged, and answers the two
// questions a command needs to ask about the adjustment.
//
// Adjust is handed the three files as separate maps, in the moment they are
// still separate — the merge that follows cannot be undone from outside. It may
// remove from any of them; what it must not do is add, because a value invented
// here would have no file behind it and `config:list --origin` would have
// nothing true to say about it.
type LayerExtension interface {
	// Adjust runs immediately before the merge. Any of the maps may be empty:
	// a project may have no machine config, and most have no installation one.
	//
	// `project` and `machine` are the extension's to change. `general` is a
	// **copy**, and it is copied rather than documented as read-only for a
	// reason: the real one is cached for the process and shared by every
	// project in it, so a removal made for one project would silently follow
	// the next — and the shared proxy generator reads several projects in one
	// run.
	Adjust(project, machine, general map[string]string)

	// Declare records that a key should be kept out on this machine, for
	// `config:unset --machine`. The error is shown to the person who typed the
	// command.
	Declare(projectName, key, activeScope string) error

	// Declared lists what has been kept out, for `config:list --origin`. An
	// adjustment nobody can see is one that gets undone by the next person.
	Declared(projectName string) []string
}

// layerExtension is the one registered extension, or nil in community.
//
// One rather than a list, deliberately: two extensions adjusting the same
// layers would have an order, and an order nobody declared is a coin flip with
// a production configuration in it.
var layerExtension LayerExtension

// RegisterLayerExtension installs the extension. Extension point for
// madock-pro; calling it twice replaces the first, which is what a test wants
// and what nothing else should do.
func RegisterLayerExtension(e LayerExtension) {
	layerExtension = e
}

// LayerExtensionRegistered reports whether anything is listening, so a command
// can say "that is madock-pro" instead of accepting a flag and doing nothing.
func LayerExtensionRegistered() bool {
	return layerExtension != nil
}

// adjustLayers gives the extension its moment. A no-op when none is registered.
func adjustLayers(project, machine, general map[string]string) {
	if layerExtension == nil {
		return
	}

	safe := make(map[string]string, len(general))
	for key, value := range general {
		safe[key] = value
	}

	layerExtension.Adjust(project, machine, safe)
}

// DeclareLayerKeyRemoved is what `config:unset --machine` calls.
func DeclareLayerKeyRemoved(projectName, key, activeScope string) error {
	if layerExtension == nil {
		return errNoLayerExtension
	}

	return layerExtension.Declare(projectName, key, activeScope)
}

// LayerKeysRemoved is what `config:list --origin` calls.
func LayerKeysRemoved(projectName string) []string {
	if layerExtension == nil {
		return nil
	}

	return layerExtension.Declared(projectName)
}

// errNoLayerExtension is the community answer, and it names the edition rather
// than the seam: "no layer extension is registered" is true and useless to the
// person reading it.
var errNoLayerExtension = layerExtensionMissing{}

type layerExtensionMissing struct{}

func (layerExtensionMissing) Error() string {
	return "keeping a key out on this machine is a madock-pro feature. " +
		"In this edition the project's committed .madock/config.xml wins every key it declares"
}
