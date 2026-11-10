package configs

import "strings"

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
func ComposeNameTakenBy(name string) string {
	return composeNameTakenBy(name, ListProjects())
}

func composeNameTakenBy(name string, entries []ProjectEntry) string {
	key := ComposeKey(name)
	for _, entry := range entries {
		if entry.Name != name && ComposeKey(entry.Name) == key {
			return entry.Name
		}
	}
	return ""
}

// ComposeNameClash is the refusal for a new project whose Compose name is
// another's, or "" when there is no clash.
func ComposeNameClash(name string) string {
	other := ComposeNameTakenBy(name)
	if other == "" {
		return ""
	}
	return "Project " + other + " already exists, and Docker Compose runs " + name + " and " + other +
		" under one name, " + NormalizeComposeName("madock_"+name) +
		": they would share containers and volumes. Give this directory another name"
}
