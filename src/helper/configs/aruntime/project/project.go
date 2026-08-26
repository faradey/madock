package project

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/dockertransform"
	"github.com/faradey/madock/v4/src/helper/embedded"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
)

// MakeConf renders the project's compose files and build context from its
// config. It runs on every up, not only on rebuild.
//
// It used to return early whenever the conf-cache marker existed, and only
// rebuild and project:clone removed that marker. So a `config:set` followed by
// `start` or `restart` kept the previously generated Dockerfile: the change was
// in config.xml, invisible to docker, and the environment ran the old one until
// somebody happened to rebuild. The same gate silently ignored a newly added
// docker-compose.<GOOS>.yml override, which the documentation promises is
// picked up on every start.
//
// Rendering is file IO over a few dozen templates and is deterministic, so
// doing it every time costs nothing worth measuring. What it must not do by
// itself is recreate containers — that decision belongs to the caller, which
// compares Fingerprint against the stack the containers were created from.
func MakeConf(projectName string) {
	// Rendering is the moment a binary older than the templates matters, so it
	// is where the warning belongs. Everything that generates a stack comes
	// through here, and nothing that merely talks to a container does — which is
	// what keeps the message off `db:execute` and `cli` in every other project on
	// a machine where somebody is editing templates.
	embedded.ReportOnce()

	// get project config
	projectConf := configs.GetProjectConfig(projectName)
	pp := paths.NewProjectPaths(projectName)
	src := paths.MakeDirsByPath(pp.RuntimeDir()) + "/src"
	if _, err := os.Lstat(src); err == nil {
		if err := os.Remove(src); err != nil {
			log.Fatalf("failed to unlink: %+v", err)
		}
	}
	err := os.Symlink(projectConf["path"], src)
	if err != nil {
		logger.Fatal(err)
	}
	makeNginxDockerfile(projectName)
	makeNginxConf(projectName)
	makeDockerCompose(projectName)
	if gen, ok := dockerConfGenerators[projectConf["platform"]]; ok {
		gen(projectName)
	}
	processOtherCTXFiles(projectName)
}

func MakeScriptsConf(projectName string) {
	exPath := paths.GetExecDirPath()
	pp := paths.NewProjectPaths(projectName)
	src := pp.CtxDir() + "/scripts"
	if fi, err := os.Lstat(src); err == nil {
		if fi.Mode()&os.ModeSymlink != os.ModeSymlink {
			err = os.RemoveAll(src)
			if err == nil {
				err = os.Symlink(exPath+"/scripts", src)
				if err != nil {
					logger.Fatal(err)
				}
			} else {
				fmt.Println(err)
			}
		}
	} else {
		err = os.Symlink(exPath+"/scripts", src)
		if err != nil {
			logger.Fatal(err)
		}
	}
}

func MakeKibanaConf(projectName string) {
	file := GetDockerConfigFile(projectName, "kibana/kibana.yml", "")
	pp := paths.NewProjectPaths(projectName)
	RenderTo(projectName, file, "kibana/kibana.yml", paths.MakeDirsByPath(pp.CtxDir())+"/kibana.yml", nil)
}

func makeNginxDockerfile(projectName string) {
	// A project that answers no request has no web server to build.
	if !configs.NginxEnabledFor(projectName) {
		return
	}
	// Platforms with a self-contained image (own nginx, e.g. packeton) ship no
	// nginx/Dockerfile — skip the unused nginx ctx instead of fataling.
	if GetDockerConfigFileOptional(projectName, "nginx/Dockerfile", "") == "" {
		return
	}
	MakeDockerfile(projectName, "nginx/Dockerfile", "nginx.Dockerfile")
}

// makeNginxConf renders the project's own nginx configuration.
//
// Which snippet is the front door — fastcgi to php, or a proxy to whatever the
// language runs — is decided by main_service, and by nothing else. The nginx
// templates used to decide it per service, one conditional include per enabled
// runtime, which emitted a server block per runtime with the same listen and
// server_name: nginx keeps the first and warns about the rest, so enabling php
// beside another runtime silently took that runtime's route away.
func makeNginxConf(projectName string) {
	if !configs.NginxEnabledFor(projectName) {
		return // the project has no web server at all
	}
	defFile := GetDockerConfigFileOptional(projectName, "nginx/conf/default.conf", "")
	if defFile == "" {
		return // platform ships no nginx conf (self-contained image)
	}

	pp := paths.NewProjectPaths(projectName)
	RenderTo(projectName, defFile, "nginx/conf/default.conf", paths.MakeDirsByPath(pp.CtxDir())+"/nginx.conf", nil)
}

func MakePhpDockerfile(projectName string) {
	makePhpDockerfiles(projectName, "php/Dockerfile", "php/DockerfileWithoutXdebug")
}

func MakeMainContainerDockerfile(projectName string) {
	projectConf := configs.GetProjectConfig(projectName)
	language := projectConf["language"]
	if language == "" {
		language = "php"
	}

	switch language {
	case "php":
		makeCustomPhpDockerfile(projectName)
	case "nodejs":
		MakeDockerfile(projectName, "Dockerfile", "nodejs.Dockerfile")
	case "python":
		MakeDockerfile(projectName, "Dockerfile", "python.Dockerfile")
	case "golang":
		MakeDockerfile(projectName, "Dockerfile", "golang.Dockerfile")
	case "ruby":
		MakeDockerfile(projectName, "Dockerfile", "ruby.Dockerfile")
	case "none":
		MakeDockerfile(projectName, "Dockerfile", "app.Dockerfile")
	default:
		makeCustomPhpDockerfile(projectName)
	}
}

func makeCustomPhpDockerfile(projectName string) {
	makePhpDockerfiles(projectName, "Dockerfile", "DockerfileWithoutXdebug")
}

// makePhpDockerfiles writes the php image and, where a platform ships one, the
// same image without xdebug. The pair differs only in which template it starts
// from: a platform with its own php/ directory keeps the first names, a custom
// project's language template keeps the second.
func makePhpDockerfiles(projectName, withXdebug, withoutXdebug string) {
	// Before anything is written, because the alternative place for this answer
	// is inside the container after a full image build — php-fpm checks the pool
	// against itself at start-up and exits if the numbers disagree.
	if err := configs.ValidateFpmPool(configs.GetProjectConfig(projectName)); err != nil {
		logger.Fatalln(err.Error())
	}

	pp := paths.NewProjectPaths(projectName)
	ctx := paths.MakeDirsByPath(pp.CtxDir())

	file := GetDockerConfigFile(projectName, withXdebug, "")
	str := Render(projectName, file, withXdebug, nil)
	write(ctx+"/php.Dockerfile", dockertransform.ApplyDockerfileTransform("php.Dockerfile", str))

	if file = GetDockerConfigFileOptional(projectName, withoutXdebug, ""); file != "" {
		str = Render(projectName, file, withoutXdebug, nil)
		write(ctx+"/php.DockerfileWithoutXdebug", dockertransform.ApplyDockerfileTransform("php.DockerfileWithoutXdebug", str))
	}
}

func write(destination, content string) {
	if err := os.WriteFile(destination, []byte(content), 0755); err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeDockerCompose(projectName string) {
	dockerDefFiles := map[string]string{
		"docker-compose.yml":          GetDockerConfigFile(projectName, "docker-compose.yml", ""),
		"docker-compose.override.yml": GetDockerConfigFileOptional(projectName, "docker-compose."+runtime.GOOS+".yml", ""),
		"docker-compose-snapshot.yml": GetDockerConfigFile(projectName, "docker-compose-snapshot.yml", "general"),
	}

	pp := paths.NewProjectPaths(projectName)
	runtimeDir := paths.MakeDirsByPath(pp.RuntimeDir())

	for name, file := range dockerDefFiles {
		// A platform without a per-OS override still gets an empty file
		// written, because compose is told to read it either way.
		str := ""
		if file != "" {
			str = Render(projectName, file, name, nil)
		}
		write(runtimeDir+"/"+name, dockertransform.ApplyComposeTransform(name, str))
	}
}

// resolveMainServiceEnabled returns "true" if the main service container will be
// emitted in docker-compose, "false" otherwise. Used to gate depends_on lines so
// that nginx (and similar) don't reference an undefined service.
// resolveMainServiceEnabled answers whether the service the front door points
// at is actually part of the stack.
//
// Every one of these services is rendered into docker-compose behind its own
// <<<if{{{<service>/enabled}}}>>>, so answering "true" for a switched-off
// service produces a file that references something it does not contain: a
// `depends_on: python` docker compose refuses to read, and an nginx upstream on
// a name that does not resolve, which stops nginx from starting. python, golang,
// ruby and app used to fall through to a blanket "true" for exactly that
// reason — only php and nodejs were answered honestly.
func resolveMainServiceEnabled(projectConf map[string]string, mainService string) string {
	switch mainService {
	case "php", "nodejs", "python", "golang", "ruby", "app":
		if projectConf[mainService+"/enabled"] == "true" {
			return "true"
		}
		return "false"
	}

	// A platform that names its own main service owns the answer.
	return "true"
}

// resolveMainServicePort returns the upstream port for the proxy.conf
// nginx template. Defaults to 3000 (Node.js/Express convention) so
// existing custom platform projects keep working; platforms with
// different conventions (Medusa = 9000, etc.) override it by writing
// main_service_port into the project config.
func resolveMainServicePort(projectConf map[string]string) string {
	if v := projectConf["main_service_port"]; v != "" {
		return v
	}
	return "3000"
}

func MakeDBDockerfile(projectName string) {
	pp := paths.NewProjectPaths(projectName)
	ctx := paths.MakeDirsByPath(pp.CtxDir())

	file := GetDockerConfigFile(projectName, "/db/Dockerfile", "")
	str := Render(projectName, file, "db/Dockerfile", nil)
	write(ctx+"/db.Dockerfile", dockertransform.ApplyDockerfileTransform("db.Dockerfile", str))

	projectConf := configs.GetProjectConfig(projectName)

	// my.cnf is only needed for MySQL/MariaDB
	if configs.GetDbType(projectConf) == "mysql" {
		str = Render(projectName, GetDockerConfigFile(projectName, "db/my.cnf", ""), "db/my.cnf", nil)

		if strings.ToLower(projectConf["db/repository"]) == "mariadb" && configs.CompareVersions(projectConf["db/version"], "10.4") >= 0 {
			str = strings.Replace(str, "[mysqld]", "[mysqld]\noptimizer_switch = 'rowid_filter=off'\noptimizer_use_condition_selectivity = 1\n", -1)
		}

		write(ctx+"/my.cnf", str)
	}
}

func MakeElasticDockerfile(projectName string) {
	MakeDockerfile(projectName, "elasticsearch/Dockerfile", "elasticsearch.Dockerfile")
}

func MakeOpenSearchDockerfile(projectName string) {
	MakeDockerfile(projectName, "opensearch/Dockerfile", "opensearch.Dockerfile")
}

func MakeRedisDockerfile(projectName string) {
	MakeDockerfile(projectName, "redis/Dockerfile", "redis.Dockerfile")
}

func MakeNodeJsDockerfile(projectName string) {
	MakeDockerfile(projectName, "nodejs/Dockerfile", "nodejs.Dockerfile")
}

func MakeClaudeDockerfile(projectName string) {
	MakeDockerfile(projectName, "claude/Dockerfile", "claude.Dockerfile")
}

func MakeDockerfile(projectName, path, fileName string) {
	file := GetDockerConfigFile(projectName, path, "")
	str := Render(projectName, file, path, nil)

	pp := paths.NewProjectPaths(projectName)
	write(paths.MakeDirsByPath(pp.CtxDir())+"/"+fileName, dockertransform.ApplyDockerfileTransform(fileName, str))
}

func GetDockerConfigFile(projectName, path, platform string) string {
	projectConf := configs.GetProjectConfig(projectName)
	if platform == "" {
		platform = projectConf["platform"]
	}
	language := projectConf["language"]
	dockerDefFile := paths.GetRunDirPath() + "/.madock/docker/" + strings.Trim(path, "/")
	if !paths.IsFileExist(dockerDefFile) {
		dockerDefFile = paths.GetExecDirPath() + "/projects/" + projectName + "/docker/" + strings.Trim(path, "/")
		if !paths.IsFileExist(dockerDefFile) {
			dockerDefFile = paths.GetExecDirPath() + "/docker/" + platform + "/" + strings.Trim(path, "/")
			if !paths.IsFileExist(dockerDefFile) {
				// Language-specific fallback (for all languages on custom platform)
				if language != "" {
					dockerDefFile = paths.GetExecDirPath() + "/docker/languages/" + language + "/" + strings.Trim(path, "/")
				}
				if !paths.IsFileExist(dockerDefFile) {
					dockerDefFile = paths.GetExecDirPath() + "/docker/general/service/" + strings.Trim(path, "/")
					if !paths.IsFileExist(dockerDefFile) {
						logger.Fatal(fmt.Errorf("docker config file not found: %s (platform=%s, language=%s)", path, platform, language))
					}
				}
			}
		}
	}

	return dockerDefFile
}

func GetDockerConfigFileOptional(projectName, path, platform string) string {
	projectConf := configs.GetProjectConfig(projectName)
	if platform == "" {
		platform = projectConf["platform"]
	}
	language := projectConf["language"]
	dockerDefFile := paths.GetRunDirPath() + "/.madock/docker/" + strings.Trim(path, "/")
	if !paths.IsFileExist(dockerDefFile) {
		dockerDefFile = paths.GetExecDirPath() + "/projects/" + projectName + "/docker/" + strings.Trim(path, "/")
		if !paths.IsFileExist(dockerDefFile) {
			dockerDefFile = paths.GetExecDirPath() + "/docker/" + platform + "/" + strings.Trim(path, "/")
			if !paths.IsFileExist(dockerDefFile) {
				if language != "" {
					dockerDefFile = paths.GetExecDirPath() + "/docker/languages/" + language + "/" + strings.Trim(path, "/")
				}
				if !paths.IsFileExist(dockerDefFile) {
					dockerDefFile = paths.GetExecDirPath() + "/docker/general/service/" + strings.Trim(path, "/")
					if !paths.IsFileExist(dockerDefFile) {
						return ""
					}
				}
			}
		}
	}

	return dockerDefFile
}

func processOtherCTXFiles(projectName string) {
	filesNames := []string{
		"grafana/loki-config.yaml",
		"grafana/promtail-config.yml",
		"grafana/prometheus-config.yml",
		"grafana/mysql-exporter.my.cnf",
		"grafana/dashboard-mysql.json",
		"grafana/dashboard-redis.json",
		"grafana/dashboard-loki.json",
	}
	pp := paths.NewProjectPaths(projectName)

	for _, fileName := range filesNames {
		// Platforms with a self-contained image (e.g. packeton) ship no grafana
		// ctx files and there is no general fallback — skip instead of fataling.
		file := GetDockerConfigFileOptional(projectName, fileName, "")
		if file == "" {
			continue
		}
		paths.MakeDirsByPath(pp.CtxDir() + "/" + strings.Split(fileName, "/")[0] + "/")
		RenderTo(projectName, file, fileName, pp.CtxDir()+"/"+fileName, nil)
	}

	ctxDir := paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/docker/ctx/")
	for _, ctxFile := range paths.GetFiles(ctxDir) {
		RenderTo(projectName, ctxDir+"/"+ctxFile, "ctx/"+ctxFile, pp.CtxDir()+"/"+ctxFile, nil)
	}
}

func GetSnippetFile(projectName, path string) string {
	snippetFile, err := FindSnippetFile(projectName, path)
	if err != nil {
		logger.Fatal(err)
	}
	return snippetFile
}

// ErrSnippetMissing is what an unresolvable include is, so a caller can tell it
// from every other way rendering fails.
//
// It exists because that difference decides whether a command may proceed:
// `rebuild` and `restart` destroy containers before they render, and a missing
// include therefore used to end the process with the environment already down.
// A template that fails for any other reason is a defect in the template; this
// one is drift, and drift can be checked for in advance.
var ErrSnippetMissing = errors.New("include not found")

// FindSnippetFile resolves an include to a file, or says where it looked.
//
// The three places, in order: the project's own override, the machine's copy for
// this project, and what the installation ships. An override survives every
// madock upgrade, so a snippet that moves in a release — `php/nodejs` became
// `common/nodejs` — leaves the override pointing at a path nothing provides, and
// nothing says so until the next build.
func FindSnippetFile(projectName, path string) (string, error) {
	looked := []string{
		paths.GetRunDirPath() + "/.madock/docker/" + strings.Trim(path, "/"),
		paths.GetExecDirPath() + "/projects/" + projectName + "/docker/" + strings.Trim(path, "/"),
		paths.GetExecDirPath() + "/docker/" + strings.Trim(path, "/"),
	}

	for _, candidate := range looked {
		if paths.IsFileExist(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("%w: %s\nLooked in:\n  %s", ErrSnippetMissing, path, strings.Join(looked, "\n  "))
}
