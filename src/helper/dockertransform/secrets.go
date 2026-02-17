package dockertransform

// DockerSecretsInjector allows enterprise to transform generated Docker files
// to replace ENV-based secrets with Docker secrets or mount-based injection.
type DockerSecretsInjector interface {
	// TransformCompose modifies docker-compose content before writing.
	TransformCompose(serviceName, content string) string
	// TransformDockerfile modifies Dockerfile content before writing.
	TransformDockerfile(serviceName, content string) string
}

var secretsInjector DockerSecretsInjector

// SetSecretsInjector sets a custom injector for Docker secrets.
func SetSecretsInjector(i DockerSecretsInjector) {
	secretsInjector = i
}

// ApplyComposeTransform applies the secrets injector to docker-compose content.
// Returns content unchanged if no injector is set.
func ApplyComposeTransform(serviceName, content string) string {
	if secretsInjector != nil {
		return secretsInjector.TransformCompose(serviceName, content)
	}
	return content
}

// ApplyDockerfileTransform applies the secrets injector to Dockerfile content.
// Returns content unchanged if no injector is set.
func ApplyDockerfileTransform(serviceName, content string) string {
	if secretsInjector != nil {
		return secretsInjector.TransformDockerfile(serviceName, content)
	}
	return content
}
