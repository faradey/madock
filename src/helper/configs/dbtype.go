package configs

import "strings"

// GetDbType determines the database type from project config.
// Returns "mysql", "postgresql", or "mongodb".
// Priority: explicit db/type > detection from db/repository > default "mysql".
func GetDbType(projectConf map[string]string) string {
	if dbType := projectConf["db/type"]; dbType != "" {
		return strings.ToLower(dbType)
	}

	return DbTypeFromRepository(projectConf["db/repository"])
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