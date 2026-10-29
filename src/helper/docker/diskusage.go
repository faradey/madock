package docker

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// Disk usage of the Docker daemon, attributed to madock's projects.
//
// Nothing in madock could answer "what is filling the Docker disk". The way to
// ask was `docker system df -v`, which madock's own rules keep a person (and an
// agent) from running beside it, and which lists hundreds of images and volumes
// by name with no notion of which project they belong to. Found when the Docker
// VM of a machine running test clones reached 88% and nobody could say whether
// the space was build cache, the clones' images or somebody's databases — and
// those three call for three different decisions.

// DiskUsage is the whole answer: totals by kind, and the same split per project.
type DiskUsage struct {
	Images     KindUsage      `json:"images"`
	Volumes    KindUsage      `json:"volumes"`
	BuildCache KindUsage      `json:"build_cache"`
	Dangling   KindUsage      `json:"dangling_images"`
	Projects   []ProjectUsage `json:"projects"`
}

// KindUsage is a count and a size. Reclaimable is what removing the unused part
// frees: images and volumes no container uses, cache entries not in use.
type KindUsage struct {
	Count       int   `json:"count"`
	Bytes       int64 `json:"bytes"`
	Reclaimable int64 `json:"reclaimable_bytes"`
}

// ProjectUsage is one madock project's share.
//
// ImageBytes counts only what is the project's own — an image's unique size —
// so a base image shared by twenty projects is not counted twenty times, and
// the sum says what removing the project's images would free.
type ProjectUsage struct {
	Project     string `json:"project"`
	Registered  bool   `json:"registered"`
	Images      int    `json:"images"`
	ImageBytes  int64  `json:"image_bytes"`
	Volumes     int    `json:"volumes"`
	VolumeBytes int64  `json:"volume_bytes"`
}

// dfReport is the part of `docker system df -v --format json` this reads.
type dfReport struct {
	Images []struct {
		ID         string `json:"ID"`
		Repository string `json:"Repository"`
		Tag        string `json:"Tag"`
		Containers string `json:"Containers"`
		Size       string `json:"Size"`
		UniqueSize string `json:"UniqueSize"`
	} `json:"Images"`
	Volumes []struct {
		Name   string `json:"Name"`
		Labels string `json:"Labels"`
		Links  string `json:"Links"`
		Size   string `json:"Size"`
	} `json:"Volumes"`
	BuildCache []struct {
		Size string `json:"Size"`
		// "true"/"false" as a string in the CLI's JSON, not a boolean.
		InUse string `json:"InUse"`
	} `json:"BuildCache"`
}

// ReadDiskUsage asks the daemon. Read-only: it lists, it removes nothing.
func ReadDiskUsage(known map[string]bool) (DiskUsage, error) {
	out, err := exec.Command("docker", "system", "df", "-v", "--format", "{{json .}}").Output()
	if err != nil {
		return DiskUsage{}, composeError(err)
	}
	return parseDiskUsage(out, known)
}

func parseDiskUsage(data []byte, known map[string]bool) (DiskUsage, error) {
	var report dfReport
	if err := json.Unmarshal(data, &report); err != nil {
		return DiskUsage{}, fmt.Errorf("docker system df answered something that is not its JSON: %w", err)
	}

	var u DiskUsage
	byProject := map[string]*ProjectUsage{}
	share := func(name string, image bool) *ProjectUsage {
		project := projectOf(name, known, image)
		if project == "" {
			return nil
		}
		p, ok := byProject[project]
		if !ok {
			p = &ProjectUsage{Project: project, Registered: known[project]}
			byProject[project] = p
		}
		return p
	}

	for _, img := range report.Images {
		// The unique size, never the full one: layers shared between images
		// would otherwise be counted once per image that has them — 249 GB of
		// <none> images on a 50 GB image store, measured on the first run.
		unique := parseSize(img.UniqueSize)
		unused := strings.TrimSpace(img.Containers) == "0"
		u.Images.Count++
		u.Images.Bytes += unique
		if unused {
			u.Images.Reclaimable += unique
		}
		if img.Repository == "<none>" {
			u.Dangling.Count++
			u.Dangling.Bytes += unique
			if unused {
				u.Dangling.Reclaimable += unique
			}
			continue
		}
		if p := share(img.Repository, true); p != nil {
			p.Images++
			p.ImageBytes += unique
		}
	}

	for _, vol := range report.Volumes {
		size := parseSize(vol.Size)
		u.Volumes.Count++
		u.Volumes.Bytes += size
		if strings.TrimSpace(vol.Links) == "0" {
			u.Volumes.Reclaimable += size
		}
		name := vol.Name
		if project := labelValue(vol.Labels, composeLabel); project != "" {
			name = project + "_"
		}
		if p := share(name, false); p != nil {
			p.Volumes++
			p.VolumeBytes += size
		}
	}

	for _, entry := range report.BuildCache {
		size := parseSize(entry.Size)
		u.BuildCache.Count++
		u.BuildCache.Bytes += size
		if !strings.EqualFold(strings.TrimSpace(entry.InUse), "true") {
			u.BuildCache.Reclaimable += size
		}
	}

	for _, p := range byProject {
		u.Projects = append(u.Projects, *p)
	}
	sort.Slice(u.Projects, func(i, j int) bool {
		a, b := u.Projects[i], u.Projects[j]
		if a.ImageBytes+a.VolumeBytes != b.ImageBytes+b.VolumeBytes {
			return a.ImageBytes+a.VolumeBytes > b.ImageBytes+b.VolumeBytes
		}
		return a.Project < b.Project
	})
	return u, nil
}

// projectOf names the madock project a resource belongs to, from a name such as
// `madock_shop-php` (image), `madock_shop_dbdata` (volume) or the compose label
// `madock_shop`, or "" when it is not madock's.
//
// The registry decides between candidates: with `shop` and `shop-update` both
// registered, `madock_shop-update-php` is shop-update's, so the longest
// registered name that fits wins. A name the registry does not know is cut the
// way compose builds it: an image is `<project>-<service>`, so at the last `-`;
// a volume is `<project>_<volume>`, so at the first `_`. It is then reported as
// unregistered.
func projectOf(name string, known map[string]bool, image bool) string {
	rest, ok := strings.CutPrefix(name, "madock_")
	if !ok || rest == "" {
		return ""
	}
	best := ""
	for project := range known {
		if len(project) <= len(best) {
			continue
		}
		if rest == project || strings.HasPrefix(rest, project+"-") || strings.HasPrefix(rest, project+"_") {
			best = project
		}
	}
	if best != "" {
		return best
	}
	if image {
		if i := strings.LastIndex(rest, "-"); i > 0 {
			return rest[:i]
		}
		return rest
	}
	if i := strings.Index(rest, "_"); i > 0 {
		return rest[:i]
	}
	return strings.TrimSuffix(rest, "_")
}

// labelValue reads one label out of docker's "k=v,k=v" rendering.
func labelValue(labels, key string) string {
	for _, pair := range strings.Split(labels, ",") {
		if k, v, ok := strings.Cut(strings.TrimSpace(pair), "="); ok && k == key {
			return v
		}
	}
	return ""
}

// parseSize reads docker's human sizes: "1.25GB", "512.3MB", "0B", "4.1kB".
// docker writes them in powers of 1000. What cannot be read counts as zero, so
// one odd entry does not take the whole report down.
func parseSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" {
		return 0
	}
	units := []struct {
		suffix string
		factor float64
	}{
		{"TB", 1e12}, {"GB", 1e9}, {"MB", 1e6}, {"kB", 1e3}, {"KB", 1e3}, {"B", 1},
	}
	for _, unit := range units {
		if number, ok := strings.CutSuffix(s, unit.suffix); ok {
			value, err := strconv.ParseFloat(strings.TrimSpace(number), 64)
			if err != nil {
				return 0
			}
			return int64(value * unit.factor)
		}
	}
	return 0
}

// DockerBytes writes a size the way docker does — powers of 1000 — so the two
// can be compared. Not the 1024-based kind madock Pro prints disk space with;
// the name says which one this is.
func DockerBytes(b int64) string {
	switch {
	case b >= 1e12:
		return fmt.Sprintf("%.2f TB", float64(b)/1e12)
	case b >= 1e9:
		return fmt.Sprintf("%.2f GB", float64(b)/1e9)
	case b >= 1e6:
		return fmt.Sprintf("%.1f MB", float64(b)/1e6)
	case b >= 1e3:
		return fmt.Sprintf("%.1f kB", float64(b)/1e3)
	}
	return fmt.Sprintf("%d B", b)
}
