package snapshotengine

import "strings"

// Volume is one named volume of a project that a snapshot carries.
//
// A snapshot used to mean the databases and the files. Everything else with
// state — the search index, the monitoring history, and since the persist keys
// Redis and RabbitMQ — stayed behind, so a clone came up with an empty index
// and needed a full reindex before its catalogue showed anything. The list is
// every named volume the project's compose declares, decided here once, so
// snapshot:create, snapshot:restore and project:clone cannot disagree about
// what a snapshot holds.
type Volume struct {
	// Name is the key under `volumes:` in the project's compose file.
	Name string
	// Database marks dbdata and dbdata2. They keep their own archives
	// (db.tar.gz, db2.tar.gz) and their own mounts in the helper container,
	// because every snapshot taken before this list existed is laid out that
	// way and has to go on restoring.
	Database bool
}

// Archive is the file a volume's tar copy is stored in, inside a snapshot
// directory.
func (v Volume) Archive() string {
	switch v.Name {
	case "dbdata":
		return "db.tar.gz"
	case "dbdata2":
		return "db2.tar.gz"
	}
	return "vol-" + v.Name + ".tar.gz"
}

// Mount is where the helper `snapshot` container sees the volume. It has to
// match docker/general/docker-compose-snapshot.yml.
func (v Volume) Mount() string {
	switch v.Name {
	case "dbdata":
		return "/var/www/mysql"
	case "dbdata2":
		return "/var/www/mysql2/mysql"
	}
	return "/var/www/volumes/" + v.Name
}

// Volumes lists the project's named volumes, in a fixed order: the databases
// first, then everything else. It has to agree with
// docker/snippets/docker-compose/volumes.yml; a test holds the two together.
//
// dbdata is listed only when the project runs its own db service — a project
// whose database lives on another project's server has no data of its own to
// copy, and archiveDatabases says so.
func Volumes(conf map[string]string) []Volume {
	on := func(key string) bool { return strings.TrimSpace(conf[key]) == "true" }
	var out []Volume
	if hasLocalDB(conf) {
		out = append(out, Volume{Name: "dbdata", Database: true})
		if on("db2/enabled") {
			out = append(out, Volume{Name: "dbdata2", Database: true})
		}
	}
	for _, engine := range []string{"elasticsearch", "opensearch", "meilisearch"} {
		if on("search/" + engine + "/enabled") {
			out = append(out, Volume{Name: engine + "_vlm_" + strings.TrimSpace(conf["search/"+engine+"/version"])})
		}
	}
	if on("grafana/enabled") {
		out = append(out, Volume{Name: "grafanadata"}, Volume{Name: "lokidata"})
	}
	for _, service := range []string{"redis", "valkey", "rabbitmq"} {
		if on(service+"/enabled") && on(service+"/persist") {
			out = append(out, Volume{Name: service + "data"})
		}
	}
	return out
}

// Others is Volumes without the databases: what an engine that takes only the
// databases leaves to tar.
func Others(conf map[string]string) []Volume {
	var out []Volume
	for _, v := range Volumes(conf) {
		if !v.Database {
			out = append(out, v)
		}
	}
	return out
}

// hasLocalDB is dbtarget.HasLocal for the one question asked here. dbtarget
// imports configs, and this package is imported by madock Pro's engines,
// which must not drag the configuration loader in with it; the rule is short
// enough to state twice, and the test pins it to dbtarget's.
func hasLocalDB(conf map[string]string) bool {
	return conf["db/enabled"] != "false"
}
