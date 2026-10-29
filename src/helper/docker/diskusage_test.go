package docker

import "testing"

// The shape `docker system df -v --format '{{json .}}'` answers with, cut down:
// sizes as human strings, InUse as a string, labels as one "k=v,k=v" string.
const dfFixture = `{
 "Images": [
  {"Repository": "madock_shop-php", "Tag": "latest", "Containers": "1", "Size": "1.2GB", "UniqueSize": "200MB"},
  {"Repository": "madock_shop-update-php", "Tag": "latest", "Containers": "0", "Size": "1.2GB", "UniqueSize": "300MB"},
  {"Repository": "madock_gone-nginx", "Tag": "latest", "Containers": "0", "Size": "50MB", "UniqueSize": "5MB"},
  {"Repository": "<none>", "Tag": "<none>", "Containers": "0", "Size": "1.2GB", "UniqueSize": "100MB"},
  {"Repository": "<none>", "Tag": "<none>", "Containers": "0", "Size": "1.2GB", "UniqueSize": "100MB"},
  {"Repository": "mariadb", "Tag": "11.4", "Containers": "2", "Size": "400MB", "UniqueSize": "400MB"}
 ],
 "Volumes": [
  {"Name": "madock_shop_dbdata", "Labels": "com.docker.compose.project=madock_shop,com.docker.compose.volume=dbdata", "Links": "1", "Size": "2GB"},
  {"Name": "madock_shop-update_dbdata", "Labels": "com.docker.compose.project=madock_shop-update", "Links": "0", "Size": "1GB"},
  {"Name": "somebodys_data", "Labels": "", "Links": "0", "Size": "3GB"}
 ],
 "BuildCache": [
  {"Size": "10GB", "InUse": "false"},
  {"Size": "1GB", "InUse": "true"}
 ]
}`

func TestParseDiskUsageSplitsByKindAndProject(t *testing.T) {
	known := map[string]bool{"shop": true, "shop-update": true}
	u, err := parseDiskUsage([]byte(dfFixture), known)
	if err != nil {
		t.Fatal(err)
	}

	// Unique sizes only: two <none> images of 1.2 GB that share all but 100 MB
	// each are 200 MB on disk, not 2.4 GB.
	if u.Dangling.Count != 2 || u.Dangling.Bytes != 200e6 {
		t.Errorf("dangling = %+v, want 2 images, 200 MB", u.Dangling)
	}
	if u.Images.Bytes != 1105e6 || u.Images.Reclaimable != 505e6 {
		t.Errorf("images = %+v, want 1105 MB, 505 MB unused", u.Images)
	}
	if u.Volumes.Bytes != 6e9 || u.Volumes.Reclaimable != 4e9 {
		t.Errorf("volumes = %+v, want 6 GB, 4 GB without a container", u.Volumes)
	}
	if u.BuildCache.Bytes != 11e9 || u.BuildCache.Reclaimable != 10e9 {
		t.Errorf("build cache = %+v, want 11 GB, 10 GB not in use", u.BuildCache)
	}

	got := map[string]ProjectUsage{}
	for _, p := range u.Projects {
		got[p.Project] = p
	}
	// shop-update's image must not be counted as shop's: the longest registered
	// name wins.
	if p := got["shop"]; p.ImageBytes != 200e6 || p.VolumeBytes != 2e9 || !p.Registered {
		t.Errorf("shop = %+v", p)
	}
	if p := got["shop-update"]; p.ImageBytes != 300e6 || p.VolumeBytes != 1e9 {
		t.Errorf("shop-update = %+v", p)
	}
	// An unregistered project's image is cut at its last dash.
	if p, ok := got["gone"]; !ok || p.Registered || p.ImageBytes != 5e6 {
		t.Errorf("gone = %+v (found %v)", p, ok)
	}
	// Somebody else's volume and a pulled image belong to no project.
	if len(got) != 3 {
		t.Errorf("projects = %v, want shop, shop-update, gone", u.Projects)
	}
}

func TestParseSize(t *testing.T) {
	cases := map[string]int64{"0B": 0, "512B": 512, "4.1kB": 4100, "1.5MB": 1500000, "2.25GB": 2250000000, "1TB": 1e12, "N/A": 0, "": 0, "junk": 0}
	for in, want := range cases {
		if got := parseSize(in); got != want {
			t.Errorf("parseSize(%q) = %d, want %d", in, got, want)
		}
	}
}
