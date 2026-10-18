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

// renderingProject is the project whose files are being generated right now.
var renderingProject string

// RenderingProject names the project whose compose file a transformer is being
// handed, or "" outside a render.
//
// The current directory's project is not the same thing, and a transformer
// that assumed it was would be wrong in exactly one command: project:clone
// generates the clone's files while standing in the source, so a transform
// reading "the current project" configures the clone as the source — for a
// volume that is the clone's database written into the source's storage.
func RenderingProject() string {
	return renderingProject
}

// ApplyProjectComposeTransform is ApplyComposeTransform for a file that belongs
// to project, which RenderingProject reports for the length of the call.
func ApplyProjectComposeTransform(project, serviceName, content string) string {
	previous := renderingProject
	renderingProject = project
	defer func() { renderingProject = previous }()
	return ApplyComposeTransform(serviceName, content)
}

// ApplyDockerfileTransform applies the transformer to Dockerfile content.
// Returns content unchanged if no transformer is set.
func ApplyDockerfileTransform(serviceName, content string) string {
	if transformer != nil {
		return transformer.TransformDockerfile(serviceName, content)
	}
	return content
}
