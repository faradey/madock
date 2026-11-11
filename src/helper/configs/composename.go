package configs

import (
	"sort"
	"strings"

	"github.com/faradey/madock/v4/src/helper/paths"
)

// ComposeKey is what Docker Compose keeps of a project's name: lower case, and
// only a-z, 0-9, '_' and '-'. Everything else is dropped, not replaced.
//
// madock writes `name: madock_<project>` into the compose file and Compose
// normalises it, so `ShopU` runs as `madock_shopu` and `Shop:V2` as
// `madock_shopv2`. Every container, volume, network and image of the project is
// named after that, and its com.docker.compose.project label is that string.
// Measured on Compose v5.7.0: `name: madock_shop+1` comes back as
// `madock_shop1`.
//
// Two projects with the same key are one Compose project: the second `start`
// recreated the first one's containers and mounted its database volume, while
// `status` of the first still said running.
func ComposeKey(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NormalizeComposeName is Compose's rule for a whole project name: ComposeKey,
// then leading '_' and '-' trimmed.
//
// Anything that looks up a project's containers, volumes or label by name has
// to go through this, not through strings.ToLower: a lookup by `madock_ShopU`
// finds nothing, and a removal that finds nothing reports success.
func NormalizeComposeName(name string) string {
	return strings.TrimLeft(ComposeKey(name), "_-")
}

// ComposeNameTakenBy names the registered project that Compose would run under
// the same name as name, or "" when there is none. name itself does not count:
// setting a project up again is not a clash with itself.
//
// A project runs under its own name in the scope default and under
// <name>-<scope> in every other scope, so project shop with a scope staging
// holds madock_shop-staging as well. Measured on a VM with 4.5.57: setup and
// start of a project shop-staging went through and took over the containers
// and the database volume shop had in staging.
func ComposeNameTakenBy(name string) string {
	return composeNameTakenBy(name, ListProjects(), registeredScopes)
}

func composeNameTakenBy(name string, entries []ProjectEntry, scopes func(string) []string) string {
	return composeKeyTakenBy(ComposeKey(name), name, entries, scopes)
}

// composeKeyTakenBy describes the run of a project other than self — its own
// name, or its name in one of its scopes — whose Compose key is key.
func composeKeyTakenBy(key, self string, entries []ProjectEntry, scopes func(string) []string) string {
	for _, entry := range entries {
		if entry.Name == self {
			continue
		}
		if ComposeKey(entry.Name) == key {
			return entry.Name
		}
		for _, scope := range scopes(entry.Name) {
			if ComposeKey(entry.Name+"-"+scope) == key {
				return entry.Name + " in scope " + scope
			}
		}
	}
	return ""
}

// ComposeNamesTakenAmong describes another project among entries that runs
// under a Compose name project runs under — its own name, or its name in one of
// its scopes — or "" when there is none. For a removal that takes everything
// under those names: with a clash, part of it is the other project's.
//
// Extension point for madock-pro, whose project:purge refuses such a project.
// Nothing in this edition removes every scope of a project, so nothing here
// calls it.
func ComposeNamesTakenAmong(project string, entries []ProjectEntry) string {
	return composeNamesTakenAmong(project, entries, registeredScopes)
}

func composeNamesTakenAmong(project string, entries []ProjectEntry, scopes func(string) []string) string {
	if other := composeKeyTakenBy(ComposeKey(project), project, entries, scopes); other != "" {
		return other
	}
	for _, scope := range scopes(project) {
		if other := composeKeyTakenBy(ComposeKey(project+"-"+scope), project, entries, scopes); other != "" {
			return other + ", the Compose name of " + project + " in scope " + scope
		}
	}
	return ""
}

// RegisteredScopes are the scopes other than default in a project's registry
// config, each a Compose project of its own, <project>-<scope>.
//
// Extension point for madock-pro, whose project:purge removes the containers,
// networks and volumes of every scope. In this edition project:orphans calls it,
// to know a scope's resources as its project's.
func RegisteredScopes(name string) []string {
	return registeredScopes(name)
}

// registeredScopes are the scopes other than default in a project's registry
// config. A file that does not parse gives none: the clash check must not end
// the process over somebody else's broken config, and its own name is still
// compared.
func registeredScopes(name string) []string {
	conf, err := ParseXmlFileSafe(paths.GetExecDirPath() + "/projects/" + name + "/config.xml")
	if err != nil {
		return nil
	}
	return scopesIn(conf)
}

func scopesIn(conf map[string]string) []string {
	seen := map[string]bool{}
	var scopes []string
	for key := range conf {
		parts := strings.Split(key, "/")
		if len(parts) < 2 || parts[0] != "scopes" || parts[1] == "default" || parts[1] == "" || seen[parts[1]] {
			continue
		}
		seen[parts[1]] = true
		scopes = append(scopes, parts[1])
	}
	sort.Strings(scopes)
	return scopes
}

// ComposeNameClash is the refusal for a new project whose Compose name is
// another's, or "" when there is no clash.
func ComposeNameClash(name string) string {
	other := ComposeNameTakenBy(name)
	if other == "" {
		return ""
	}
	return "Project " + other + " already runs under " + NormalizeComposeName("madock_"+name) +
		", the Docker Compose name " + name + " would get: they would share containers and volumes. Give this directory another name"
}

// ScopeComposeClash is the refusal for a scope of project whose Compose name,
// <project>-<scope>, another project already runs under, or "" when there is
// no clash.
func ScopeComposeClash(project, scope string) string {
	return scopeComposeClash(project, scope, ListProjects(), registeredScopes)
}

func scopeComposeClash(project, scope string, entries []ProjectEntry, scopes func(string) []string) string {
	other := composeKeyTakenBy(ComposeKey(project+"-"+scope), project, entries, scopes)
	if other == "" {
		return ""
	}
	return "Project " + other + " already runs under " + NormalizeComposeName("madock_"+project+"-"+scope) +
		", the Docker Compose name of " + project + " in scope " + scope +
		": they would share containers and volumes. Choose another scope name"
}
