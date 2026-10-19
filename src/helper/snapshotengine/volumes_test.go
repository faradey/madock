package snapshotengine

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	dockerassets "github.com/faradey/madock/v4/docker"
	"github.com/faradey/madock/v4/src/helper/dbtarget"
)

func names(vs []Volume) []string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.Name)
	}
	return out
}

// everything is a project with every volume-bearing service switched on.
func everything() map[string]string {
	return map[string]string{
		"db2/enabled":                  "true",
		"search/elasticsearch/enabled": "true",
		"search/elasticsearch/version": "8.4.3",
		"search/opensearch/enabled":    "true",
		"search/opensearch/version":    "2.19.1",
		"search/meilisearch/enabled":   "true",
		"search/meilisearch/version":   "1.11.3",
		"grafana/enabled":              "true",
		"redis/enabled":                "true",
		"redis/persist":                "true",
		"valkey/enabled":               "true",
		"valkey/persist":               "true",
		"rabbitmq/enabled":             "true",
		"rabbitmq/persist":             "true",
	}
}

func TestVolumesOfADefaultProjectAreTheDatabaseOnly(t *testing.T) {
	got := names(Volumes(map[string]string{}))
	if strings.Join(got, ",") != "dbdata" {
		t.Fatalf("a project with defaults should carry dbdata alone, got %v", got)
	}
	if len(Others(map[string]string{})) != 0 {
		t.Fatalf("a project with defaults has no volumes beyond the database")
	}
}

// Redis switched on without persist has no named volume: listing it would send
// tar to a mount that does not exist.
func TestAServiceWithoutPersistIsNotListed(t *testing.T) {
	conf := map[string]string{"redis/enabled": "true", "rabbitmq/enabled": "true", "rabbitmq/persist": "false"}
	if got := names(Others(conf)); len(got) != 0 {
		t.Fatalf("services without persist were listed: %v", got)
	}
	conf = map[string]string{"redis/persist": "true"}
	if got := names(Others(conf)); len(got) != 0 {
		t.Fatalf("persist on a disabled service was listed: %v", got)
	}
}

func TestEveryVolumeIsListedWhenEverythingIsOn(t *testing.T) {
	got := strings.Join(names(Volumes(everything())), ",")
	want := "dbdata,dbdata2,elasticsearch_vlm_8.4.3,opensearch_vlm_2.19.1,meilisearch_vlm_1.11.3,grafanadata,lokidata,redisdata,valkeydata,rabbitmqdata"
	if got != want {
		t.Fatalf("volumes:\n got %s\nwant %s", got, want)
	}
}

// Old snapshots hold db.tar.gz and db2.tar.gz at those names and restore into
// those mounts; renaming either would strand every one of them.
func TestTheDatabasesKeepTheirArchivesAndMounts(t *testing.T) {
	for _, c := range []struct{ name, archive, mount string }{
		{"dbdata", "db.tar.gz", "/var/www/mysql"},
		{"dbdata2", "db2.tar.gz", "/var/www/mysql2/mysql"},
		{"opensearch_vlm_2.19.1", "vol-opensearch_vlm_2.19.1.tar.gz", "/var/www/volumes/opensearch_vlm_2.19.1"},
	} {
		v := Volume{Name: c.name}
		if v.Archive() != c.archive || v.Mount() != c.mount {
			t.Errorf("%s: archive %s mount %s, want %s %s", c.name, v.Archive(), v.Mount(), c.archive, c.mount)
		}
	}
}

func TestNoDatabaseVolumeWithoutALocalDatabase(t *testing.T) {
	for _, conf := range []map[string]string{
		{"db/enabled": "false", "db2/enabled": "true"},
		{},
		{"db/enabled": "true"},
	} {
		listed := len(Volumes(conf)) > 0 && Volumes(conf)[0].Name == "dbdata"
		if listed != dbtarget.HasLocal(conf, "db") {
			t.Errorf("db/enabled=%q: listed dbdata %v, dbtarget.HasLocal %v", conf["db/enabled"], listed, dbtarget.HasLocal(conf, "db"))
		}
	}
}

var templateVolumeKey = regexp.MustCompile(`(?m)^  ([a-z0-9_]+(?:\{\{\{\.search\.[a-z]+\.version\}\}\})?):\s*$`)

// declared reads the top-level volume keys a compose template can produce,
// with the search version placeholders kept as written.
func declared(t *testing.T, file string) []string {
	t.Helper()
	data, err := dockerassets.FS.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if i := strings.Index(text, "\nvolumes:"); i >= 0 {
		text = text[i:]
	}
	var out []string
	for _, m := range templateVolumeKey.FindAllStringSubmatch(text, -1) {
		out = append(out, m[1])
	}
	sort.Strings(out)
	return out
}

// The project's compose, the helper container's compose and Volumes are three
// lists of the same thing. If volumes.yml gains a volume the other two do not
// know, a snapshot silently leaves it behind — the bug this list was written to
// end — so they are held together here.
func TestTheListMatchesBothComposeTemplates(t *testing.T) {
	var fromGo []string
	for _, n := range names(Volumes(everything())) {
		n = strings.Replace(n, "_vlm_8.4.3", "_vlm_{{{.search.elasticsearch.version}}}", 1)
		n = strings.Replace(n, "_vlm_2.19.1", "_vlm_{{{.search.opensearch.version}}}", 1)
		n = strings.Replace(n, "_vlm_1.11.3", "_vlm_{{{.search.meilisearch.version}}}", 1)
		fromGo = append(fromGo, n)
	}
	sort.Strings(fromGo)
	want := strings.Join(fromGo, ",")

	for _, file := range []string{"snippets/docker-compose/volumes.yml", "general/docker-compose-snapshot.yml"} {
		if got := strings.Join(declared(t, file), ","); got != want {
			t.Errorf("%s declares\n %s\nVolumes lists\n %s", file, got, want)
		}
	}

	snapshot, _ := dockerassets.FS.ReadFile("general/docker-compose-snapshot.yml")
	for _, v := range Others(everything()) {
		mount := v.Mount()
		name := v.Name
		for _, ver := range [][2]string{{"8.4.3", "{{{.search.elasticsearch.version}}}"}, {"2.19.1", "{{{.search.opensearch.version}}}"}, {"1.11.3", "{{{.search.meilisearch.version}}}"}} {
			name = strings.Replace(name, ver[0], ver[1], 1)
			mount = strings.Replace(mount, ver[0], ver[1], 1)
		}
		if !strings.Contains(string(snapshot), "- "+name+":"+mount) {
			t.Errorf("the helper container does not mount %s at %s", name, mount)
		}
	}
}

type fakeEngine struct{ live, all bool }

func (f fakeEngine) Live() bool                              { return f.live }
func (f fakeEngine) Capture(Project, string) (string, error) { return "", nil }
func (f fakeEngine) Restore(Project, string) error           { return nil }
func (f fakeEngine) Clone(Project, Project) error            { return nil }
func (f fakeEngine) Drop(Project, string) error              { return nil }

type fakeAllEngine struct{ fakeEngine }

func (f fakeAllEngine) AllVolumes() bool { return f.all }

// A live engine that takes only the databases must still stop a project with a
// search index: the index goes through tar, and tar of a running OpenSearch is
// torn.
func TestAVolumeLeftToTarStopsTheProject(t *testing.T) {
	search := map[string]string{"search/opensearch/enabled": "true", "search/opensearch/version": "2.19.1"}
	if Stops(fakeEngine{live: true}, map[string]string{}) {
		t.Errorf("a live engine stopped a project that has nothing but its database")
	}
	if !Stops(fakeEngine{live: true}, search) {
		t.Errorf("a live engine left the search index to tar and did not stop the project")
	}
	if Stops(fakeAllEngine{fakeEngine{live: true, all: true}}, search) {
		t.Errorf("an engine that takes every volume still stopped the project")
	}
	if !Stops(fakeAllEngine{fakeEngine{live: true, all: false}}, search) {
		t.Errorf("AllVolumes() false was taken as true")
	}
}
