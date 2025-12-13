package ports

import (
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
)

const (
	BasePort    = 17000
	MaxPort     = 65535
	PortsFile   = "/aruntime/ports.conf"
	NextPortKey = "__next__"
)

// Service names for port allocation
const (
	ServiceNginx       = "nginx"
	ServiceNginxSSL    = "nginx_ssl"
	ServiceDB          = "db"
	ServiceDB2         = "db2"
	ServiceLiveReload  = "livereload"
	ServiceVite        = "vite"
)

// Registry holds the port allocations
type Registry struct {
	ports    map[string]int
	nextPort int
	filePath string
}

// NewRegistry creates a new port registry
func NewRegistry() *Registry {
	r := &Registry{
		ports:    make(map[string]int),
		nextPort: BasePort,
		filePath: paths.GetExecDirPath() + PortsFile,
	}
	r.load()
	return r
}

// load reads the ports.conf file
func (r *Registry) load() {
	if !paths.IsFileExist(r.filePath) {
		return
	}

	content, err := os.ReadFile(r.filePath)
	if err != nil {
		logger.Fatal(err)
	}

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

		port, err := strconv.Atoi(value)
		if err != nil {
			continue
		}

		if key == NextPortKey {
			r.nextPort = port
		} else {
			r.ports[key] = port
		}
	}

	// If nextPort wasn't set, calculate it from existing ports
	if r.nextPort == BasePort && len(r.ports) > 0 {
		maxPort := BasePort
		for _, port := range r.ports {
			if port >= maxPort {
				maxPort = port + 1
			}
		}
		r.nextPort = maxPort
	}
}

// save writes the ports.conf file
func (r *Registry) save() {
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime")

	// Sort keys for consistent output
	var keys []string
	for key := range r.ports {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var lines []string
	lines = append(lines, "# Port allocations (do not edit manually)")
	for _, key := range keys {
		lines = append(lines, key+"="+strconv.Itoa(r.ports[key]))
	}
	lines = append(lines, NextPortKey+"="+strconv.Itoa(r.nextPort))

	content := strings.Join(lines, "\n") + "\n"
	err := os.WriteFile(r.filePath, []byte(content), 0664)
	if err != nil {
		logger.Fatal(err)
	}
}

// GetOrAllocate returns existing port or allocates a new one
func (r *Registry) GetOrAllocate(projectName, serviceName string) int {
	key := projectName + "/" + serviceName

	if port, exists := r.ports[key]; exists {
		return port
	}

	// Allocate new port
	port := r.nextPort
	r.ports[key] = port
	r.nextPort++
	r.save()

	return port
}

// Get returns port for a service, 0 if not found
func (r *Registry) Get(projectName, serviceName string) int {
	key := projectName + "/" + serviceName
	return r.ports[key]
}

// GetAllForProject returns all ports for a project
func (r *Registry) GetAllForProject(projectName string) map[string]int {
	result := make(map[string]int)
	prefix := projectName + "/"

	for key, port := range r.ports {
		if strings.HasPrefix(key, prefix) {
			serviceName := strings.TrimPrefix(key, prefix)
			result[serviceName] = port
		}
	}

	return result
}

// RemoveProject removes all ports for a project
func (r *Registry) RemoveProject(projectName string) {
	prefix := projectName + "/"
	for key := range r.ports {
		if strings.HasPrefix(key, prefix) {
			delete(r.ports, key)
		}
	}
	r.save()
}

// IsOldFormat checks if ports.conf is in old format (project=number)
func (r *Registry) IsOldFormat() bool {
	if !paths.IsFileExist(r.filePath) {
		return false
	}

	content, err := os.ReadFile(r.filePath)
	if err != nil {
		return false
	}

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
		// Old format has no "/" in keys (except __next__)
		if key != NextPortKey && !strings.Contains(key, "/") {
			return true
		}
	}

	return false
}

// MigrateFromOldFormat converts old format to new format
func (r *Registry) MigrateFromOldFormat() {
	if !paths.IsFileExist(r.filePath) {
		return
	}

	content, err := os.ReadFile(r.filePath)
	if err != nil {
		return
	}

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

		portNum, err := strconv.Atoi(value)
		if err != nil {
			continue
		}

		// Skip if already new format
		if strings.Contains(key, "/") || key == NextPortKey {
			continue
		}

		oldPorts[key] = portNum
	}

	if len(oldPorts) == 0 {
		return
	}

	// Convert old format to new format
	// Old formula: basePort = 17000 + (portNum - 1) * 12
	maxPort := BasePort
	newPorts := make(map[string]int)

	for projectName, portNum := range oldPorts {
		basePort := BasePort + (portNum-1)*12

		newPorts[projectName+"/"+ServiceNginx] = basePort + 0
		newPorts[projectName+"/"+ServiceNginxSSL] = basePort + 1
		newPorts[projectName+"/"+ServiceDB] = basePort + 2
		newPorts[projectName+"/"+ServiceDB2] = basePort + 3
		newPorts[projectName+"/"+ServiceLiveReload] = basePort + 4
		newPorts[projectName+"/"+ServiceVite] = basePort + 5

		if basePort+11 > maxPort {
			maxPort = basePort + 11
		}
	}

	r.ports = newPorts
	r.nextPort = maxPort + 1
	r.save()
}

// Global registry instance
var globalRegistry *Registry

// GetRegistry returns the global port registry
func GetRegistry() *Registry {
	if globalRegistry == nil {
		globalRegistry = NewRegistry()
	}
	return globalRegistry
}

// GetPort is a convenience function to get or allocate a port
func GetPort(projectName, serviceName string) int {
	return GetRegistry().GetOrAllocate(projectName, serviceName)
}
