package configs

// NginxEnabled answers whether a project gets a web server at all.
//
// Most do, and the default is true. What this is for is the project that
// answers no request and never will: the owner of a shared database schema, a
// queue worker, a bus consumer. extmag-core-shopify is the first — it holds the
// tables and the shop tokens for a cluster of Shopify apps, each of which takes
// its own webhooks, so Core has no HTTP process left at all.
//
// Removing the project's <hosts> was not enough and was worse than nothing: the
// container still started, and the shared proxy still wrote a server block for
// it, because a project with no hosts gets loc.<name>.com invented for it — so
// the block was renamed rather than removed, and the ports stayed reserved.
//
// **"The ports" is true of a project that ships with this off and false of one
// switched off later, and the difference is not a bug.** The value is normally
// committed in the project's own .madock/config.xml — extmag-core-shopify is
// the case — and that file is read before the first render, so nothing ever
// asks for a web port. `config:set nginx/enabled false` on a running project
// arrives after `setup` has rendered and allocated, and nothing hands a number
// back except `project:remove`. Both paths are measured in
// test/e2e/nginx_optional_test.go.
//
// Releasing them on the switch was considered and rejected on 2026-09-08.
// Ports run 17000–65535 and a removal already frees a project's whole set, so
// the leak is at most two numbers per project that was switched off while
// running — against the cost of numbers that move under a project people have
// bookmarked, scripted or written into a firewall rule.
//
// A missing key is true, which is what an installation upgrading into this
// change looks like before anything rewrites its project configurations.
func NginxEnabled(conf map[string]string) bool {
	return conf["nginx/enabled"] != "false"
}

// NginxEnabledFor is NginxEnabled for a project by name, for the callers that
// have not already read its configuration.
func NginxEnabledFor(projectName string) bool {
	return NginxEnabled(GetProjectConfig(projectName))
}
