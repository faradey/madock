package dockertransform

// DockerTransformer allows enterprise to transform generated Docker files
// (Dockerfiles and docker-compose) before they are written to disk.
type DockerTransformer interface {
	// TransformCompose modifies docker-compose content before writing.
	TransformCompose(serviceName, content string) string
	// TransformDockerfile modifies Dockerfile content before writing.
	TransformDockerfile(serviceName, content string) string
}

var transformer DockerTransformer

// SetDockerTransformer installs the transformer applied to generated Docker
// files.
//
// Extension point for madock-pro, which uses it to keep secrets out of the
// files it renders. Unreachable from this module by design.
func SetDockerTransformer(t DockerTransformer) {
	transformer = t
}

// Deprecated: Use SetDockerTransformer instead.
func SetSecretsInjector(i DockerTransformer) {
	SetDockerTransformer(i)
}

// ApplyComposeTransform applies the transformer to docker-compose content.
// Returns content unchanged if no transformer is set.
func ApplyComposeTransform(serviceName, content string) string {
	if transformer != nil {
		return transformer.TransformCompose(serviceName, content)
	}
	return content
}

// ApplyDockerfileTransform applies the transformer to Dockerfile content.
// Returns content unchanged if no transformer is set.
func ApplyDockerfileTransform(serviceName, content string) string {
	if transformer != nil {
		return transformer.TransformDockerfile(serviceName, content)
	}
	return content
}