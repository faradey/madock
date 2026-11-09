package install

import "strings"

// SearchInstallArgs may add arguments to Magento's `setup:install` for the
// search engine: the login, for one that asks for a password. It gets the
// project configuration, the flag family the command uses for the engine
// ("opensearch" or "elasticsearch" — Magento before 2.4.6 reaches OpenSearch
// through the elasticsearch flags) and the service that runs it
// ("opensearch" or "elasticsearch"). It returns whole arguments, such as
// "--opensearch-password=…"; each is shell-quoted here.
//
// Extension point for madock-pro, which puts passwords on the search engines.
// The install is one shell line run in the php container, so arguments added
// after it would come too late.
type SearchInstallArgs func(projectConf map[string]string, flagFamily, service string) []string

var searchInstallArgsProviders []SearchInstallArgs

// RegisterSearchInstallArgs adds a provider; their arguments follow the
// search flags in registration order.
func RegisterSearchInstallArgs(p SearchInstallArgs) {
	searchInstallArgsProviders = append(searchInstallArgsProviders, p)
}

// searchInstallArgs renders the registered providers' arguments, each quoted
// and followed by a space. With none registered it is empty, and the install
// line is the one madock has always run.
func searchInstallArgs(projectConf map[string]string, flagFamily, service string) string {
	var b strings.Builder
	for _, p := range searchInstallArgsProviders {
		for _, arg := range p(projectConf, flagFamily, service) {
			b.WriteString(shellSingleQuote(arg))
			b.WriteString(" ")
		}
	}
	return b.String()
}
