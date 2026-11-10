package docker

import (
	"strings"

	"github.com/faradey/madock/v4/src/helper/configs"
)

// GetContainerName returns the full container name for a service: Compose's
// own name for the project, then the service and the replica. Lower-casing
// alone was not Compose's rule — a project called shop+1 runs as madock_shop1,
// and a name with the '+' in it named no container at all.
func GetContainerName(projectConf map[string]string, projectName, service string) string {
	scope := ""
	if val, ok := projectConf["activeScope"]; ok && val != "default" {
		scope = strings.ToLower("-" + val)
	}
	return configs.NormalizeComposeName(projectConf["container_name_prefix"]+projectName+scope) + "-" + service + "-1"
}
