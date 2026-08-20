package configs

import "strings"

// GetDbType determines the database type from project config.
// Returns "mysql", "postgresql", or "mongodb".
// Priority: explicit db/type > detection from db/repository > default "mysql".
//
// The explicit value is normalized, and that is not tidiness. db/type names an
// engine *family* — it is what every compose snippet gates on
// (`eq .db.type "mysql"`, "postgresql", "mongodb") — while db/repository names
// the image. "mariadb" is a repository wearing the family's name, and a config
// carrying it matched no snippet at all: the generated compose came out with no
// `db` service, `start` reported success, `status` listed php and nginx and said
// nothing about a database, and `info` cheerfully printed "type MARIADB, host
// db". The application then failed somewhere else entirely — on Magento,
// `bin/magento` answering "There are no commands defined in the ... namespace",
// a symptom that leads nowhere near the cause.
//
// Found in the field on a project whose config madock itself wrote in March
// 2024, so the radius is every configuration old enough to predate the current
// writer, plus every hand-edited .madock/config.xml. Normalizing at read time
// rather than migrating the file covers both, and covers them without a madock
// command writing into a config that belongs to the project's repository.
func GetDbType(projectConf map[string]string) string {
	if dbType := projectConf["db/type"]; strings.TrimSpace(dbType) != "" {
		if normalized, ok := normalizeDbType(dbType); ok {
			return normalized
		}

		// Unrecognized: fall through to the repository rather than default
		// blindly to mysql. A typo next to `db/repository=postgres:16` still
		// gets postgres, and the worst case is the same "mysql" the fallback
		// ends at anyway. What must never happen again is an answer that
		// matches no snippet, because that one is silent.
	}

	return DbTypeFromRepository(projectConf["db/repository"])
}

// normalizeDbType maps what a config may say to the three families the compose
// snippets gate on. Reports false for anything it does not recognize, so the
// caller can decide — deliberately not "mysql", which would make a typo and a
// deliberate choice indistinguishable here.
func normalizeDbType(dbType string) (string, bool) {
	switch value := strings.ToLower(strings.TrimSpace(dbType)); {
	case value == "mysql", value == "mariadb", value == "maria", value == "percona":
		return "mysql", true
	case strings.HasPrefix(value, "postgres"), value == "pgsql", value == "psql":
		return "postgresql", true
	case strings.HasPrefix(value, "mongo"):
		return "mongodb", true
	default:
		return "", false
	}
}

// DbTypeFromRepository reads the engine out of an image name.
//
// Separate from GetDbType because it is also needed where db/type is being
// decided rather than read: `--db postgres:16` names a repository and a
// version, and the engine written beside it used to come from the interactive
// question instead — which, under --yes, nobody answered. A project then had
// db/repository=postgres with db/type=mysql, so the generated compose picked
// the MySQL service block, handed it the postgres image and set MYSQL_* on it.
func DbTypeFromRepository(repository string) string {
	repo := strings.ToLower(repository)
	switch {
	case strings.HasPrefix(repo, "postgres"):
		return "postgresql"
	case strings.HasPrefix(repo, "mongo"):
		return "mongodb"
	default:
		return "mysql"
	}
}
