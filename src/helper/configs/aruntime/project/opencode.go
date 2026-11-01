package project

import (
	"os"
	"path/filepath"

	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/opencode"
	"github.com/faradey/madock/v4/src/helper/paths"
)

// makeOpencode writes what `madock opencode` needs, or removes it when the
// project has opencode off. Removal matters as much as writing: the files are
// mounted only while the services exist, and one left behind from an earlier
// configuration is a sandbox's instructions nobody is looking at. The keys
// stay — a key that changed on every toggle would gain nothing.
//
// See src/helper/opencode for why the sandbox has the shape it has.
func makeOpencode(projectName string) {
	projectConf := configs.GetProjectConfig(projectName)
	ctx := paths.MakeDirsByPath(paths.NewProjectPaths(projectName).CtxDir())
	files := []string{opencode.ConfigCtxFile, opencode.AgentsCtxFile, opencode.SshdCtxFile, opencode.LLMCtxFile, "opencode.Dockerfile"}

	if !opencode.Enabled(projectConf) {
		for _, name := range files {
			if err := os.Remove(filepath.Join(ctx, name)); err != nil && !os.IsNotExist(err) {
				logger.Fatalln("removing " + filepath.Join(ctx, name) + ": " + err.Error())
			}
		}
		return
	}

	if err := opencode.Version(projectConf); err != nil {
		logger.Fatalln(err.Error())
	}
	conf, err := opencode.Config(projectConf)
	if err != nil {
		logger.Fatalln(err.Error())
	}
	agents, err := opencode.Agents(projectConf)
	if err != nil {
		logger.Fatalln(err.Error())
	}
	llm, err := opencode.LLMConf(projectConf["opencode/base_url"])
	if err != nil {
		logger.Fatalln(err.Error())
	}
	if err := opencode.EnsureKeys(paths.NewProjectPaths(projectName).RuntimeDir()); err != nil {
		logger.Fatalln("opencode keys: " + err.Error())
	}

	MakeDockerfile(projectName, "opencode/Dockerfile", "opencode.Dockerfile")
	for name, content := range map[string]string{
		opencode.ConfigCtxFile: conf,
		opencode.AgentsCtxFile: agents,
		opencode.SshdCtxFile:   opencode.SshdConf(),
		opencode.LLMCtxFile:    llm,
	} {
		if err := os.WriteFile(filepath.Join(ctx, name), []byte(content), 0644); err != nil {
			logger.Fatalln("writing " + filepath.Join(ctx, name) + ": " + err.Error())
		}
	}
}

// opencodeCompose adds the opencode services to docker-compose.yml.
func opencodeCompose(projectName, fileName, content string) string {
	projectConf := configs.GetProjectConfig(projectName)
	if fileName != "docker-compose.yml" || !opencode.Enabled(projectConf) {
		return content
	}
	compose := opencode.InjectCompose(content, configs.ResolveMainService(projectConf, "php"))
	if compose == content {
		// The main service was not in the file, so there is nothing to build
		// opencode-exec from. Said rather than skipped: `madock opencode`
		// would otherwise fail later on a container that was never defined.
		logger.Fatalln("opencode/enabled is on, but the project's main service is not in docker-compose.yml; opencode needs it to run the project's tools")
	}
	return compose
}

// opencodeDockerfile adds sshd to an application image, for opencode-exec.
func opencodeDockerfile(projectName, fileName, content string) string {
	projectConf := configs.GetProjectConfig(projectName)
	if !opencode.Enabled(projectConf) || !opencode.MainDockerfile(configs.ResolveMainService(projectConf, "php"), fileName) {
		return content
	}
	return opencode.InjectDockerfile(content)
}
