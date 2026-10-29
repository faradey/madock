package docker

import "testing"

// What a cleanup may remove. A wrong answer here removes an image somebody
// else's stack needs, or one a stopped madock project still runs from.
func TestPlanFromTakesOnlyUnusedImagesOfMadockProjects(t *testing.T) {
	df := []byte(`{
 "Images": [
  {"ID": "aaaaaaaaaaaa", "Repository": "madock_shop-php", "Tag": "latest", "Containers": "0", "UniqueSize": "300MB"},
  {"ID": "bbbbbbbbbbbb", "Repository": "madock_shop-db", "Tag": "latest", "Containers": "1", "UniqueSize": "200MB"},
  {"ID": "cccccccccccc", "Repository": "<none>", "Tag": "<none>", "Containers": "0", "UniqueSize": "100MB"},
  {"ID": "dddddddddddd", "Repository": "<none>", "Tag": "<none>", "Containers": "0", "UniqueSize": "900MB"},
  {"ID": "eeeeeeeeeeee", "Repository": "someones-api", "Tag": "1", "Containers": "0", "UniqueSize": "50MB"}
 ],
 "Volumes": [{"Name": "madock_shop_dbdata", "Links": "0", "Size": "5GB"}],
 "BuildCache": [{"Size": "10GB", "InUse": "false"}, {"Size": "2GB", "InUse": "true"}]
}`)
	labelled := []labelledImage{
		// An untagged image a madock rebuild left: ours only by its label.
		{id: "cccccccccccc", name: "cccccccccccc", project: "madock_shop"},
		// Another compose stack's untagged image: not ours.
		{id: "dddddddddddd", name: "dddddddddddd", project: "ci-runner"},
		// A used one is kept whatever its label says.
		{id: "bbbbbbbbbbbb", name: "madock_shop-db:latest", project: "madock_shop"},
	}

	plan, err := planFrom(df, labelled)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]bool{}
	for _, img := range plan.Images {
		got[img.ID] = true
	}
	if !got["aaaaaaaaaaaa"] || !got["cccccccccccc"] || len(got) != 2 {
		t.Errorf("images = %+v, want the unused madock_shop-php and the madock-labelled <none>", plan.Images)
	}
	if plan.ImageBytes != 400e6 {
		t.Errorf("image bytes = %d, want 400 MB", plan.ImageBytes)
	}
	if plan.BuildCacheBytes != 10e9 {
		t.Errorf("build cache = %d, want the 10 GB not in use", plan.BuildCacheBytes)
	}
}

// `image inspect` is where the label is; `image ls` cannot print it.
func TestParseLabelledImages(t *testing.T) {
	out := "sha256:aaaaaaaaaaaa1111\tmadock_shop\tmadock_shop-php:latest,madock_shop-php:old\n" +
		"sha256:cccccccccccc2222\tmadock_shop\t\n"
	got := parseLabelledImages(out)
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got[0] != (labelledImage{id: "aaaaaaaaaaaa", name: "madock_shop-php:latest", project: "madock_shop"}) {
		t.Errorf("first = %+v", got[0])
	}
	// An untagged image is named by its ID, so a report can still say which.
	if got[1] != (labelledImage{id: "cccccccccccc", name: "cccccccccccc", project: "madock_shop"}) {
		t.Errorf("second = %+v", got[1])
	}
}

// Without --all docker keeps every reusable cache entry, and the removal fell
// short of the plan by 41.7 GB of 60.0.
func TestBuildCachePruneTakesEveryUnusedEntry(t *testing.T) {
	has := false
	for _, arg := range pruneBuildCacheArgs() {
		if arg == "--all" {
			has = true
		}
	}
	if !has {
		t.Fatalf("builder prune without --all removes only dangling entries: %v", pruneBuildCacheArgs())
	}
}
