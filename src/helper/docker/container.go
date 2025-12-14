package docker

import "strings"

// GetContainerName returns the full container name for a service
func GetContainerName(projectConf map[string]string, projectName, service string) string {
	scope := ""
	if val, ok := projectConf["activeScope"]; ok && val != "default" {
		scope = strings.ToLower("-" + val)
	}
	return strings.ToLower(projectConf["container_name_prefix"]) + strings.ToLower(projectName) + scope + "-" + service + "-1"
}
