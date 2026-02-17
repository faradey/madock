package project

// dockerConfGenerators maps platform name to its docker configuration generator.
var dockerConfGenerators = map[string]func(string){}

// RegisterDockerConfGenerator registers a function that generates
// platform-specific Docker configuration files (Dockerfiles, configs, etc.)
// for the given platform name.
func RegisterDockerConfGenerator(platform string, fn func(string)) {
	dockerConfGenerators[platform] = fn
}
