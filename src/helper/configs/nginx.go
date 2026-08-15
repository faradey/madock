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
