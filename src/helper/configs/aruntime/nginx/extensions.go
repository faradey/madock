package nginx

// Extension points in the shared proxy configuration.
//
// The shared proxy is one file for every project on a machine, and the parts of
// it that belong to the paid edition — anything about what the proxy trusts, or
// how it counts a client — are registered from there rather than compiled in
// here. Same seam as configs.SetSecretsProvider: this repository declares where
// the text goes and nothing about what it says.
//
// Nothing in this repository registers an extension, so the list is empty in
// community and the generated file is exactly what it was before the seam
// existed.

// PreambleExtension renders text for the top of the http block, above the rate
// limit and connection zones and above the log format.
//
// It is given the installation-wide configuration because that is what the
// preamble is built from: the shared proxy has no project of its own.
type PreambleExtension func(generalConfig map[string]string) string

var preambleExtensions []PreambleExtension

// RegisterPreambleExtension adds text to the top of the http block.
//
// Extension point for madock-pro. Order of registration is order of output, and
// a registration that renders nothing costs nothing — an extension decides for
// itself whether the configuration asks for it.
func RegisterPreambleExtension(e PreambleExtension) {
	preambleExtensions = append(preambleExtensions, e)
}

// PreambleExtensions renders every registered extension, in order.
func PreambleExtensions(generalConfig map[string]string) string {
	out := ""
	for _, e := range preambleExtensions {
		out += e(generalConfig)
	}
	return out
}
