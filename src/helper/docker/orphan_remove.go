package docker

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/faradey/madock/v4/src/helper/configs"
)

// The removal half of project:orphans: what is left of one forgotten project,
// and the refusals that stand between that and `docker volume rm`.
//
// It exists because the listing alone left a server with no way to act on it.
// The leftovers are not in the registry, so `project:remove` cannot reach them,
// and the advice was `docker volume rm` by hand — on a machine where madock was
// asked to be the only thing that touches containers. A demo host that removes
// a stand's directory every day accumulates exactly these.
//
// What it may remove is decided twice, and both must agree. The refusals below
// look at the name before anything is listed, and the plan is built from the
// same rule that decides what project:orphans lists, so a name the registry
// holds yields nothing to remove even if a refusal were taken out.

// OrphanRemoval is what `project:orphans --remove` takes away for one project.
type OrphanRemoval struct {
	Project  string       `json:"project"`
	Volumes  []string     `json:"volumes"`
	Networks []string     `json:"networks"`
	Images   []PruneImage `json:"images"`
}

// Empty reports whether nothing labelled with the project is left.
func (r OrphanRemoval) Empty() bool {
	return len(r.Volumes) == 0 && len(r.Networks) == 0 && len(r.Images) == 0
}

// OrphanRemovalRefusal says why the resources labelled madock_<project> may not
// be removed, or "" when they may. known is the set of names the registry holds,
// as for FindOrphans.
//
// Refused: a name not written as project:orphans prints it, a label a
// registered project runs under in any of its scopes, and a label any
// container — running or stopped — still carries. A docker that cannot be
// asked is a refusal too: "no containers" is not something to assume before a
// volume is deleted.
func OrphanRemovalRefusal(project string, known map[string]bool) string {
	return orphanRemovalRefusal(project, known, configs.RegisteredScopesRead, containersLabelled)
}

func orphanRemovalRefusal(project string, known map[string]bool, scopes func(string) ([]string, bool), containers func(label string) ([]string, error)) string {
	// As printed, because that is the only spelling that is certain. Compose
	// drops what it does not keep, so `Shop,V2` would quietly become shopv2 — a
	// different project than the one typed — and a removal is the last place to
	// guess which one was meant.
	if project == "" || configs.ComposeKey(project) != project {
		return "'" + project + "' is not a name project:orphans prints. Give it exactly as listed there: lower case, a-z, 0-9, '_' and '-'"
	}

	if owner := readRegistryKeys(known, scopes).takenBy(project); owner != "" {
		return "madock_" + project + " is the Docker Compose name of the registered project " + owner +
			". Its volumes are that project's data, not leftovers; project:remove is the command for a registered project"
	}

	label := "madock_" + project
	names, err := containers(label)
	if err != nil {
		return "Could not ask docker which containers carry " + label + ", so nothing is removed: " + err.Error()
	}
	if len(names) > 0 {
		return "Containers still carry " + label + ": " + strings.Join(names, ", ") +
			". Something runs under this name, or did and kept its containers; its volumes are not left behind. " +
			"Stop and remove those containers first if they are really nobody's"
	}

	return ""
}

// containersLabelled names every container, running or stopped, whose compose
// project label is label.
func containersLabelled(label string) ([]string, error) {
	out, err := exec.Command("docker", "ps", "-a",
		"--filter", "label="+composeLabel+"="+label,
		"--format", "{{.Names}}").Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps: %w", err)
	}
	return strings.Fields(string(out)), nil
}

// PlanOrphanRemoval lists what is left of project on this machine. It removes
// nothing. Call it after OrphanRemovalRefusal has answered "".
func PlanOrphanRemoval(project string, known map[string]bool) OrphanRemoval {
	var images []string
	ids := map[string]PruneImage{}
	for _, img := range labelledImages() {
		images = append(images, img.id+"\t"+img.project)
		ids[img.id] = PruneImage{ID: img.id, Name: img.name}
	}
	return orphanRemovalPlan(project,
		dockerLines("volume", "ls"),
		dockerLines("network", "ls"),
		images, ids, known, configs.RegisteredScopesRead)
}

// orphanRemovalPlan keeps, of what docker listed, the orphans of project — by
// orphansOf, the rule project:orphans lists by, and nothing looser. Images
// arrive as "<id>\t<label>" and are looked up in ids.
func orphanRemovalPlan(project string, volumes, networks, images []string, ids map[string]PruneImage, known map[string]bool, scopes func(string) ([]string, bool)) OrphanRemoval {
	plan := OrphanRemoval{Project: project}
	for _, o := range orphansOf("volume", volumes, known, scopes) {
		if o.Project == project {
			plan.Volumes = append(plan.Volumes, o.Name)
		}
	}
	for _, o := range orphansOf("network", networks, known, scopes) {
		if o.Project == project {
			plan.Networks = append(plan.Networks, o.Name)
		}
	}
	for _, o := range orphansOf("image", images, known, scopes) {
		if o.Project == project {
			plan.Images = append(plan.Images, ids[o.Name])
		}
	}
	return plan
}

// RemoveOrphans removes a plan: volumes, then networks, then images, each on
// its own, and says what docker refused. A refusal is reported, never forced —
// a volume a container took since the plan was read is in use, and that is the
// answer.
func RemoveOrphans(plan OrphanRemoval) (removed int, refused []string) {
	for _, name := range plan.Volumes {
		if out, err := exec.Command("docker", "volume", "rm", name).CombinedOutput(); err != nil {
			refused = append(refused, "volume "+name+": "+lastLines(string(out), 1))
			continue
		}
		removed++
	}
	for _, name := range plan.Networks {
		if out, err := exec.Command("docker", "network", "rm", name).CombinedOutput(); err != nil {
			refused = append(refused, "network "+name+": "+lastLines(string(out), 1))
			continue
		}
		removed++
	}
	gone, failed := RemoveImages(plan.Images)
	removed += gone
	for _, line := range failed {
		refused = append(refused, "image "+line)
	}
	return removed, refused
}
