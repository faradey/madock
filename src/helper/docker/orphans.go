package docker

import (
	"os/exec"
	"sort"
	"strings"
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
// **Read-only, and deliberately so.** Removing what this finds stays with
// `project:remove`, behind `allow_destructive_commands`. A `--remove` flag here
// would be a second route to deleting volumes that does not pass that switch,
// and the switch is the thing that stops a habit from becoming an accident.
//
// `known` is the set of project names the registry holds, including the ones it
// can no longer read: a broken registry entry is somebody's to fix, not an
// orphan, and `project:list --stale` is where it belongs.
func FindOrphans(known map[string]bool) []Orphan {
	var out []Orphan

	out = append(out, orphansOf("volume", dockerLines("volume", "ls"), known)...)
	out = append(out, orphansOf("network", dockerLines("network", "ls"), known)...)
	out = append(out, orphansOf("image", dockerLines("image", "ls"), known)...)

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
	// Found writing system:prune, which asked the same question and got the
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
// shown their own leftovers or somebody else's containers.
func orphansOf(kind string, lines []string, known map[string]bool) []Orphan {
	var out []Orphan

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
		if known[project] {
			continue
		}

		out = append(out, Orphan{Kind: kind, Name: name, Project: project})
	}

	return out
}
