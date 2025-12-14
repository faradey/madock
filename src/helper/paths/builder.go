package paths

import "path/filepath"

// ProjectPaths provides path building for project-specific paths
type ProjectPaths struct {
	projectName string
}

// NewProjectPaths creates a new ProjectPaths instance
func NewProjectPaths(projectName string) *ProjectPaths {
	return &ProjectPaths{projectName: projectName}
}

// RuntimeDir returns the project runtime directory
// Example: /path/to/madock/aruntime/projects/myproject
func (p *ProjectPaths) RuntimeDir() string {
	return filepath.Join(GetExecDirPath(), "aruntime", "projects", p.projectName)
}

// DockerCompose returns the docker-compose.yml path
func (p *ProjectPaths) DockerCompose() string {
	return filepath.Join(p.RuntimeDir(), "docker-compose.yml")
}

// DockerComposeOverride returns the docker-compose.override.yml path
func (p *ProjectPaths) DockerComposeOverride() string {
	return filepath.Join(p.RuntimeDir(), "docker-compose.override.yml")
}

// DockerComposeSnapshot returns the docker-compose-snapshot.yml path
func (p *ProjectPaths) DockerComposeSnapshot() string {
	return filepath.Join(p.RuntimeDir(), "docker-compose-snapshot.yml")
}

// ComposerDir returns the project composer directory
func (p *ProjectPaths) ComposerDir() string {
	return filepath.Join(p.RuntimeDir(), "composer")
}

// SSHDir returns the project ssh directory
func (p *ProjectPaths) SSHDir() string {
	return filepath.Join(p.RuntimeDir(), "ssh")
}

// Global paths (not project-specific)

// RuntimeBase returns the aruntime base directory
func RuntimeBase() string {
	return filepath.Join(GetExecDirPath(), "aruntime")
}

// RuntimeProjects returns the aruntime/projects directory
func RuntimeProjects() string {
	return filepath.Join(RuntimeBase(), "projects")
}

// ProxyDockerCompose returns the proxy docker-compose.yml path
func ProxyDockerCompose() string {
	return filepath.Join(RuntimeBase(), "docker-compose.yml")
}

// CtxDir returns the aruntime/ctx directory
func CtxDir() string {
	return filepath.Join(RuntimeBase(), "ctx")
}

// ComposerDir returns the aruntime/.composer directory
func ComposerDir() string {
	return filepath.Join(RuntimeBase(), ".composer")
}

// CacheDir returns the cache directory
func CacheDir() string {
	return filepath.Join(GetExecDirPath(), "cache")
}

// CtxPath returns the project ctx directory
func (p *ProjectPaths) CtxDir() string {
	return filepath.Join(p.RuntimeDir(), "ctx")
}

// StoppedFile returns the stopped marker file path
func (p *ProjectPaths) StoppedFile() string {
	return filepath.Join(p.RuntimeDir(), "stopped")
}
