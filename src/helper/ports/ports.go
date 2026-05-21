package ports

import (
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
)

const (
	BasePort  = 17000
	MaxPort   = 65535
	PortsFile = "/aruntime/ports.conf"
)

// Service names for port allocation
const (
	ServiceNginx              = "nginx"              // +0
	ServiceNginxSSL           = "nginx_ssl"          // +1
	ServicePhpMyAdmin         = "phpmyadmin"         // +2
	ServiceKibana             = "kibana"             // +3
	ServiceDB                 = "db"                 // +4
	ServiceLiveReload         = "livereload"         // +5
	ServiceDB2                = "db2"                // +6
	ServicePhpMyAdmin2        = "phpmyadmin2"        // +7
	ServiceSelenium           = "selenium"           // +8
	ServiceVarnish            = "varnish"            // +9
	ServiceGrafana            = "grafana"            // +10
	ServiceVite               = "vite"               // +11
	ServiceRabbitMQ              = "rabbitmq"              // +12
	ServiceRabbitMQManagement    = "rabbitmq_management"    // +13
	ServiceOpenSearchDashboard   = "opensearchdashboard"    // +14
)

// Registry holds the port allocations
type Registry struct {
	ports    map[string]int
	filePath string
}

// NewRegistry creates a new port registry
func NewRegistry() *Registry {
	r := &Registry{
		ports:    make(map[string]int),
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

		// Only load new format entries (project/service=port)
		if strings.Contains(key, "/") {
			r.ports[key] = port
		}
	}
}

// save writes the ports.conf file
func (r *Registry) save() {
	paths.MakeDirsByPath(paths.RuntimeBase())

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

	// Find first available port starting from BasePort
	port := r.findAvailablePort()
	r.ports[key] = port
	r.save()

	return port
}

// findAvailablePort finds the first port that is:
//   - unused in this registry,
//   - not currently bound on the host (active listener), and
//   - not claimed by any docker container's port mapping (even stopped
//     containers reserve their published ports for their next start).
//
// The triple check guards against the multi-binary scenario: a second
// madock installation running from a different exec_dir keeps its own
// ports.conf, and a project from that other installation may be stopped
// at the moment we allocate — the host-bind probe would say the port is
// free, but the moment the other project starts again it collides. The
// docker scan covers those quiescent reservations.
func (r *Registry) findAvailablePort() int {
	usedPorts := make(map[int]bool)
	for _, port := range r.ports {
		usedPorts[port] = true
	}

	dockerClaimed := dockerClaimedPorts()

	for port := BasePort; port < MaxPort; port++ {
		if usedPorts[port] {
			continue
		}
		if dockerClaimed[port] {
			continue
		}
		if !isHostPortFree(port) {
			continue
		}
		return port
	}

	// Fallback (should never happen)
	return BasePort
}

// dockerClaimedPorts returns the set of host ports that any docker
// container (running or stopped) has declared in its port mappings.
// Stopped containers still hold those reservations until removed —
// `docker ps --format {{.Ports}}` shows an empty column for them, so
// we read HostConfig.PortBindings via `docker inspect` instead, which
// returns the configured bindings regardless of run state.
// Returns an empty set when docker is unavailable — in that case we
// simply skip this check and rely on the host-bind probe.
func dockerClaimedPorts() map[int]bool {
	claimed := make(map[int]bool)

	idsOut, err := exec.Command("docker", "ps", "-aq").Output()
	if err != nil {
		return claimed
	}
	ids := strings.Fields(strings.TrimSpace(string(idsOut)))
	if len(ids) == 0 {
		return claimed
	}

	args := append([]string{
		"inspect",
		"--format",
		"{{range $port, $b := .HostConfig.PortBindings}}{{range $b}}{{.HostPort}}\n{{end}}{{end}}",
	}, ids...)
	out, err := exec.Command("docker", args...).Output()
	if err != nil {
		return claimed
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if p, err := strconv.Atoi(line); err == nil {
			claimed[p] = true
		}
	}
	return claimed
}

// isHostPortFree returns true if the given TCP port is not currently
// bound on the local host. We try listening on both IPv4 and IPv6
// because docker may bind on either family.
func isHostPortFree(port int) bool {
	addrs := []string{"0.0.0.0:" + strconv.Itoa(port), "[::]:" + strconv.Itoa(port)}
	for _, addr := range addrs {
		l, err := net.Listen("tcp", addr)
		if err != nil {
			return false
		}
		// Close immediately; we only needed the binding probe.
		// SO_REUSEADDR will let docker grab the same port a moment later.
		_ = l.Close()
	}
	// Short delay so the kernel fully releases the socket before
	// docker tries to bind it from the next compose up.
	time.Sleep(10 * time.Millisecond)
	return true
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

// Set sets a specific port for a service (used by migration)
func (r *Registry) Set(projectName, serviceName string, port int) {
	key := projectName + "/" + serviceName
	r.ports[key] = port
}

// Save persists the registry to disk
func (r *Registry) Save() {
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

// ResetRegistry clears the global registry so it will be re-initialized on next use.
func ResetRegistry() {
	globalRegistry = nil
}

// GetPort is a convenience function to get or allocate a port
func GetPort(projectName, serviceName string) int {
	return GetRegistry().GetOrAllocate(projectName, serviceName)
}
