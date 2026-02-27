package configs

import "strings"

// GetDbType determines the database type from project config.
// Returns "mysql", "postgresql", or "mongodb".
// Priority: explicit db/type > detection from db/repository > default "mysql".
func GetDbType(projectConf map[string]string) string {
	if dbType := projectConf["db/type"]; dbType != "" {
		return strings.ToLower(dbType)
	}

	repo := strings.ToLower(projectConf["db/repository"])
	switch {
	case repo == "postgres" || repo == "postgresql" || strings.HasPrefix(repo, "postgres"):
		return "postgresql"
	case repo == "mongo" || repo == "mongodb" || strings.HasPrefix(repo, "mongo"):
		return "mongodb"
	default:
		return "mysql"
	}
}