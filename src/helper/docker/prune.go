package docker

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// What a cleanup of the Docker disk may remove, and nothing else. madock has no
// command for it — madock Pro's `disk:clean` is the one, and it uses these, so
// there is one answer to "which images are madock's" and one way to empty the
// build cache. A `system:prune` here briefly duplicated it and was taken out.
//
// Two kinds, chosen because neither holds data and both come back on their own:
//
//   - images of madock projects that no container uses, running or stopped —
//     the next `start` pulls or builds them again;
//   - the build cache — the next build fills it again.
//
// Volumes are never candidates. They are databases, and removing one is
// project:remove's, behind allow_destructive_commands, with the project named.
//
// The images are madock's only: an image counts when its compose label names a
// madock project, or, for one without the label, when its name starts with
// `madock_`. The build cache cannot be told apart that way — Docker keeps one
// for the whole machine — and is removed only when asked for by name, and said
// to be the machine's.

// PruneImage is one image a cleanup would remove.
type PruneImage struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
}

// PrunePlan is what a prune would free, read before anything is removed.
type PrunePlan struct {
	Images          []PruneImage `json:"images"`
	ImageBytes      int64        `json:"image_bytes"`
	BuildCacheBytes int64        `json:"build_cache_bytes"`
}

// PlanPrune reads what is removable. It removes nothing.
func PlanPrune() (PrunePlan, error) {
	df, err := exec.Command("docker", "system", "df", "-v", "--format", "{{json .}}").Output()
	if err != nil {
		return PrunePlan{}, composeError(err)
	}
	return planFrom(df, labelledImages())
}

// planFrom is the decision on the two answers, testable without a daemon.
func planFrom(df []byte, labelled []labelledImage) (PrunePlan, error) {
	var report dfReport
	if err := json.Unmarshal(df, &report); err != nil {
		return PrunePlan{}, fmt.Errorf("docker system df answered something that is not its JSON: %w", err)
	}

	ours := map[string]bool{}
	for _, img := range labelled {
		if isMadockProject(img.project) {
			ours[img.id] = true
		}
	}

	var plan PrunePlan
	for _, img := range report.Images {
		// Used by any container, running or stopped: a stopped project starts
		// from it, and docker would refuse anyway.
		if strings.TrimSpace(img.Containers) != "0" {
			continue
		}
		mine := ours[shortID(img.ID)] || strings.HasPrefix(img.Repository, "madock_")
		if !mine {
			continue
		}
		name := img.Repository
		if img.Tag != "" && img.Tag != "<none>" {
			name += ":" + img.Tag
		}
		size := parseSize(img.UniqueSize)
		plan.Images = append(plan.Images, PruneImage{ID: shortID(img.ID), Name: name, Bytes: size})
		plan.ImageBytes += size
	}
	for _, entry := range report.BuildCache {
		if !strings.EqualFold(strings.TrimSpace(entry.InUse), "true") {
			plan.BuildCacheBytes += parseSize(entry.Size)
		}
	}
	return plan, nil
}

// shortID is the 12 characters `docker system df` prints, whatever form the ID
// arrived in.
func shortID(id string) string {
	id = strings.TrimPrefix(strings.TrimSpace(id), "sha256:")
	if len(id) > 12 {
		id = id[:12]
	}
	return id
}

// RemoveImages removes the images of a plan, by ID, and returns how many went.
// An image another one is built on, or one a container took since the plan was
// read, is refused by docker and stays; that is reported, not forced.
func RemoveImages(images []PruneImage) (removed int, refused []string) {
	for _, img := range images {
		if out, err := exec.Command("docker", "image", "rm", img.ID).CombinedOutput(); err != nil {
			refused = append(refused, img.Name+" ("+img.ID+"): "+lastLines(string(out), 1))
			continue
		}
		removed++
	}
	return removed, refused
}

// PruneBuildCache removes the build cache no build is using, for the whole
// machine.
//
// `--all`, because without it docker removes only the dangling entries — those
// no image refers to — and keeps every reusable layer. The plan counts all
// entries not in use, and the two disagreed by most of the cache: on the first
// run 60.0 GB were planned, 18.3 GB went, and `system:df` still showed 41.7 GB
// "unused" afterwards.
//
// The output is returned for its last line, `Total: 41.72GB`, which is what
// docker says it freed.
func PruneBuildCache() (string, error) {
	out, err := exec.Command("docker", pruneBuildCacheArgs()...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("docker builder prune: %w: %s", err, lastLines(string(out), 3))
	}
	return string(out), nil
}

// pruneBuildCacheArgs is separate so the flag that makes the removal match the
// plan can be asserted.
func pruneBuildCacheArgs() []string {
	return []string{"builder", "prune", "--all", "--force"}
}
