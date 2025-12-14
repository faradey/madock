package versions

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/ports"
)

const (
	basePort = 17000
)

// V310 migrates ports.conf from old format (project=number) to new format (project/service=port)
func V310() {
	portsFile := paths.GetExecDirPath() + "/aruntime/ports.conf"
	if !paths.IsFileExist(portsFile) {
		return
	}

	content, err := os.ReadFile(portsFile)
	if err != nil {
		return
	}

	// Check if migration needed (old format has no "/" in keys)
	oldPorts := make(map[string]int)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Skip if already new format
		if strings.Contains(key, "/") {
			continue
		}

		portNum, err := strconv.Atoi(value)
		if err != nil {
			continue
		}

		oldPorts[key] = portNum
	}

	if len(oldPorts) == 0 {
		return
	}

	// Create backup before migration
	backupFile := portsFile + ".backup." + time.Now().Format("20060102-150405")
	os.WriteFile(backupFile, content, 0664)

	// Convert old format to new format
	// Old formula: basePort = 17000 + (portNum - 1) * 12
	registry := ports.GetRegistry()
	for projectName, portNum := range oldPorts {
		portBase := basePort + (portNum-1)*12

		registry.Set(projectName, ports.ServiceNginx, portBase+0)
		registry.Set(projectName, ports.ServiceNginxSSL, portBase+1)
		registry.Set(projectName, ports.ServiceDB, portBase+2)
		registry.Set(projectName, ports.ServiceDB2, portBase+3)
		registry.Set(projectName, ports.ServiceLiveReload, portBase+4)
		registry.Set(projectName, ports.ServiceVite, portBase+5)
	}
	registry.Save()
}
