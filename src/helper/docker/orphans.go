package docker

import (
	"os/exec"
	"sort"
	"strings"

	"github.com/faradey/madock/v4/src/helper/configs"
)

// Orphan is one docker resource whose compose project no longer has an entry in
// madock's registry.
type Orphan struct {
	// Kind is volume, network or image.
	Kind string `json:"kind"`
	Name string `json:"name"`
	// Project is the madock project name, with the compose prefix stripped —
	// the name somebody would type, not the label docker stores.
	Project string `json:"project"`
}

// composeLabel is what `docker compose up` writes on everything it creates, and
// the only thing that still connects a volume to the project it belonged to
// once the project's directory is gone.
const composeLabel = "com.docker.compose.project"

// FindOrphans reports docker resources left behind by projects the registry no
// longer knows.
//
// It exists because nothing could answer the question. A project's volume
// survives its directory, and after that no madock command mentions it: the
// volume is not in the registry, `project:list` reads the registry, and
// `project:remove` works from a registry entry there is none of. On a machine
// where destructive commands are switched off — which is every server, as
// shipped — the only cleanup available was to delete the directories and leave
// the volumes, which turns visible litter into invisible litter.
//
// **Read-only.** Removing what this finds is `project:orphans --remove`, one
// project at a time and behind `allow_destructive_commands` like
// `project:remove`: a removal that did not pass that switch would be a second
// route to deleting volumes, and the switch is the thing that stops a habit
// from becoming an accident.
//
// `known` is the set of project names the registry holds, including the ones it
// can no longer read: a broken registry entry is somebody's to fix, not an
// orphan, and `project:list --stale` is where it belongs.
func FindOrphans(known map[string]bool) []Orphan {
	var out []Orphan

	out = append(out, orphansOf("volume", dockerLines("volume", "ls"), known, configs.RegisteredScopesRead)...)
	out = append(out, orphansOf("network", dockerLines("network", "ls"), known, configs.RegisteredScopesRead)...)
	out = append(out, orphansOf("image", dockerLines("image", "ls"), known, configs.RegisteredScopesRead)...)

	sort.Slice(out, func(i, j int) bool {
		if out[i].Project != out[j].Project {
			return out[i].Project < out[j].Project
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Name < out[j].Name
	})

	return out
}

// dockerLines asks docker for everything of one kind that carries the compose
// label, as "<name>\t<project label>" per line.
//
// A failure is an empty list rather than an error: this command reports what it
// can see, and a docker that cannot be asked is a machine where there is
// nothing to report anyway — every other madock command will have said so
// first, and louder.
func dockerLines(kind, verb string) []string {
	// `docker image ls` has no .Name and no labels at all in its template —
	// neither .Label nor .Labels — so the shared format failed for images and,
	// a failure being an empty list, project:orphans never reported an image.
	// Found writing an image cleanup, which asked the same question and got the
	// template error. The labels are only in `image inspect`.
	if kind == "image" {
		var lines []string
		for _, img := range labelledImages() {
			lines = append(lines, img.name+"\t"+img.project)
		}
		return lines
	}

	cmd := exec.Command("docker", kind, verb,
		"--filter", "label="+composeLabel,
		"--format", "{{.Name}}\t{{.Label \""+composeLabel+"\"}}")

	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	return strings.Split(strings.TrimSpace(string(out)), "\n")
}

// labelledImage is an image carrying the compose project label.
type labelledImage struct {
	id      string // the 12-character short form
	name    string // first repo:tag, or the ID for an untagged image
	project string // the label's value, "madock_<name>" for ours
}

// labelledImages lists the images with a compose project label: the IDs from
// `image ls`, which can filter on a label but not print one, and the labels from
// one `image inspect` of all of them. Nil when docker cannot be asked.
func labelledImages() []labelledImage {
	ids, err := exec.Command("docker", "image", "ls", "-a", "-q", "--no-trunc", "--filter", "label="+composeLabel).Output()
	if err != nil {
		return nil
	}
	list := strings.Fields(string(ids))
	if len(list) == 0 {
		return nil
	}
	seen := map[string]bool{}
	args := []string{"image", "inspect", "--format", "{{.Id}}\t{{index .Config.Labels \"" + composeLabel + "\"}}\t{{join .RepoTags \",\"}}"}
	for _, id := range list {
		if !seen[id] {
			seen[id] = true
			args = append(args, id)
		}
	}
	out, err := exec.Command("docker", args...).Output()
	if err != nil {
		return nil
	}
	return parseLabelledImages(string(out))
}

func parseLabelledImages(out string) []labelledImage {
	var images []labelledImage
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.SplitN(strings.TrimSpace(line), "\t", 3)
		if len(fields) < 2 || fields[0] == "" {
			continue
		}
		img := labelledImage{id: shortID(fields[0]), project: fields[1]}
		if len(fields) == 3 && fields[2] != "" {
			img.name, _, _ = strings.Cut(fields[2], ",")
		} else {
			img.name = img.id
		}
		images = append(images, img)
	}
	return images
}

// orphansOf keeps the resources whose project is one of madock's and is not in
// the registry.
//
// Split from the docker call so the rule can be tested without a daemon, which
// matters more here than usual: the rule is what decides whether somebody is
// shown their own leftovers or somebody else's containers. scopes gives the
// scopes other than default of a registered project, and false when its
// registry config cannot be read, as RegisteredScopesRead does.
func orphansOf(kind string, lines []string, known map[string]bool, scopes func(string) ([]string, bool)) []Orphan {
	var out []Orphan

	registry := readRegistryKeys(known, scopes)

	for _, line := range lines {
		name, label, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok || name == "" || label == "" {
			continue
		}

		// Only what madock made. Another compose project on the same machine —
		// somebody's own stack, a CI runner's — carries this label too, and
		// listing it would be reporting a stranger's disk as our mess.
		project, isOurs := strings.CutPrefix(label, "madock_")
		if !isOurs || project == "" {
			continue
		}
		if registry.takenBy(project) != "" {
			continue
		}

		out = append(out, Orphan{Kind: kind, Name: name, Project: project})
	}

	return out
}

// registryKeys is the registry as Compose sees it.
type registryKeys struct {
	// owners is, per Compose key a registered project runs under, who runs
	// under it: the project's name, or its name in one of its scopes.
	owners map[string]string
	// readable is, per registered project's own Compose key, whether its scopes
	// are known. Two names with one key are one Compose project, and one
	// unreadable config among them makes the key's scopes unknown.
	readable map[string]bool
}

// readRegistryKeys is known, the names the registry holds, in Compose's terms.
//
// The label is Compose's name for the project, not the registry's: project
// ShopU is labelled madock_shopu. Compared as written, every live resource of a
// project with a capital letter was listed as an orphan, with `docker volume rm`
// suggested for its database.
//
// A project runs as <name>-<scope> in every scope but default, so shop in scope
// staging is labelled madock_shop-staging. Without its scopes, that scope's live
// database was listed as the orphan of a project shop-staging, with the same
// suggestion.
func readRegistryKeys(known map[string]bool, scopes func(string) ([]string, bool)) registryKeys {
	// Sorted, so that when two names share a key the one a refusal names does
	// not change from run to run.
	names := make([]string, 0, len(known))
	for name := range known {
		names = append(names, name)
	}
	sort.Strings(names)

	r := registryKeys{owners: map[string]string{}, readable: map[string]bool{}}
	for _, name := range names {
		key := configs.ComposeKey(name)
		if _, taken := r.owners[key]; !taken {
			r.owners[key] = name
		}
		scopeNames, ok := scopes(name)
		if prev, seen := r.readable[key]; seen {
			ok = ok && prev
		}
		r.readable[key] = ok
		for _, scope := range scopeNames {
			scopeKey := configs.ComposeKey(name + "-" + scope)
			if _, taken := r.owners[scopeKey]; !taken {
				r.owners[scopeKey] = name + " in scope " + scope
			}
		}
	}
	return r
}

// takenBy describes the registered project that runs, or may run, under the
// label madock_<project>, or "" when none does.
func (r registryKeys) takenBy(project string) string {
	if owner, ok := r.owners[project]; ok {
		return owner
	}
	if key := unreadableScopeOf(project, r.readable); key != "" {
		return r.owners[key] + " in a scope its registry config.xml does not say, because it cannot be read"
	}
	return ""
}

// unreadableScopeOf answers whether project, a label with madock_ stripped, may
// be a scope of a registered project whose scopes cannot be read — it is
// <key>-<anything> for such a project's key — and returns that key, or "".
//
// A registry config that does not parse gives no scopes, and the live
// resources of that project's scopes were listed as the orphans of a project
// <name>-<scope>. The entry is counted as known for the same reason a broken
// entry is: somebody's to fix, and not with `docker volume rm`.
//
// The registered key nearest the label decides, longest first: with shop
// unreadable and shop-x registered, madock_shop-x-gone is judged by shop-x's
// own scopes, not swallowed by shop. The '-' is part of the match, so
// madock_shopx is never shop's.
func unreadableScopeOf(project string, readable map[string]bool) string {
	for i := strings.LastIndex(project, "-"); i > 0; i = strings.LastIndex(project[:i], "-") {
		if ok, isRegistered := readable[project[:i]]; isRegistered {
			if ok {
				return ""
			}
			return project[:i]
		}
	}
	return ""
}
