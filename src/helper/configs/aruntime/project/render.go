package project

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/ports"
	"github.com/faradey/madock/v3/src/helper/tmpl"
)

// This file is the whole seam between madock and its template engine.
//
// It used to be spread across every generator: each one read a file, spliced
// its includes, ran the config substitution, and then patched in by hand
// whatever the general pass could not answer — the main service before the
// conditionals could see it, the hosts joined into a string with the YAML
// indentation of the file they were going into, the ports allocated by a regex.
// The order of those steps was load-bearing and documented in an eight-line
// comment explaining why one of them had to run early.
//
// Now there is one function. Everything a template can ask for is decided here,
// once, before anything is rendered.

// Render renders a template file that has already been located, and stops the
// program if it cannot. Generation happens on every `madock start`, so a
// template that does not render is not something to carry on past: the old
// engine did carry on, and wrote an nginx config with six server blocks where
// one belonged.
func Render(projectName, file, name string, extra map[string]string) string {
	body, err := os.ReadFile(file)
	if err != nil {
		logger.Fatal(err)
	}

	out, err := newRenderer(projectName, file, extra).Render(name, string(body))
	if err != nil {
		logger.Fatal(fmt.Errorf("rendering %s for project %s: %w%s", name, projectName, err, adviceFor(err)))
	}

	return out
}

// adviceFor names the cause and the cure when the error is one whose message
// points at the wrong thing.
//
// `function "memShareGB" not defined` reads as a broken template and sends
// somebody to edit a file that is correct. It is not: template functions are
// compiled into the binary while the templates are files on disk, so the
// message means the installed binary is older than the tree it is rendering.
// Measured on this machine — the function arrived in the templates at 16:13 and
// the binary in use had been built at 13:44 the same day — and the radius there
// is every project at once, because the installation directory and the source
// checkout are the same directory: `docker/` moves with git and the binary does
// not.
//
// Written as a suffix rather than folded into the error so the original text
// survives verbatim; the point is to add what is missing, not to paraphrase
// what the engine said.
func adviceFor(err error) string {
	if err == nil || !strings.Contains(err.Error(), "not defined") {
		return ""
	}

	return "\n\nThe function is compiled into madock, the template is a file on disk," +
		"\nso this means the binary in use is older than the templates it is rendering." +
		"\nRebuild or reinstall madock; the template needs no editing."
}

// RenderTo renders a located template and writes the result where the
// generators put it: mode 0755, directories already made by the caller.
func RenderTo(projectName, file, name, destination string, extra map[string]string) string {
	out := Render(projectName, file, name, extra)
	if err := os.WriteFile(destination, []byte(out), 0755); err != nil {
		logger.Fatal(fmt.Errorf("writing %s: %w", destination, err))
	}
	return out
}

func newRenderer(projectName, file string, extra map[string]string) *tmpl.Renderer {
	conf := configs.GetProjectConfig(projectName)

	// Which file a snippet name came from, so a warning about one can name it.
	legacySource := map[string]string{}

	return &tmpl.Renderer{
		Values: renderValues(projectName, conf, extra),
		Data:   renderData(projectName, conf),
		Snippet: func(name string) (string, error) {
			snippet := GetSnippetFile(projectName, name)
			legacySource[name] = snippet
			body, err := os.ReadFile(snippet)
			return string(body), err
		},
		Port: func(service string) (int, error) {
			return ports.GetPort(projectName, service), nil
		},
		OnLegacy: func(name string, notes []string) {
			source := file
			if snippet, ok := legacySource[name]; ok {
				source = snippet
			}
			legacyWarning(source, notes)
		},
	}
}

// renderValues is the project's configuration plus everything a template may
// ask about that no configuration file holds.
//
// The four fake keys are gone from here, and that is the point of the exercise:
// db/type_is_mysql, db/type_is_postgresql, db/type_is_mongodb and
// db/use_default_auth_plugin were booleans Go computed and wrote into the config
// map as though a user had set them, because a template could not compare two
// values. A template now writes the comparison itself.
// legacyWarning names the file on disk and the command that fixes it.
//
// Naming the template — "nginx/conf/default.conf" — told the reader which
// template it was and not which file: an override lives under
// <PROJECT>/.madock/docker/ and the shipped copy under the installation, and
// only one of them is theirs to edit. And the instruction used to be "rewrite
// it", against a syntax reference, when the conversion this very message just
// performed can be written to the file by one command.
func legacyWarning(path string, notes []string) {
	fmtc.WarningLn("The template " + path + " is written in the old <<<if>>> syntax and was converted as it was read.")
	for _, note := range notes {
		fmtc.WarningLn("  " + note)
	}
	fmtc.ToDoLn("Convert it for good with: madock template:convert " + filepath.Dir(path))
	fmtc.WarningLn("The conversion on read goes away in a later release.")
}

func renderValues(projectName string, conf map[string]string, extra map[string]string) map[string]string {
	values := make(map[string]string, len(conf)+16)
	for key, value := range conf {
		values[key] = value
	}

	// db/type is a real setting with a real fallback — an unset type is read
	// off the repository name — so what a template sees is the answer, not the
	// raw field.
	values["db/type"] = configs.GetDbType(conf)

	values["project_name"] = strings.ToLower(projectName)
	values["scope"] = configs.GetActiveScope(projectName, false, "-")

	mainService := configs.ResolveMainService(conf, "php")
	values["main_service"] = mainService
	values["main_service_enabled"] = resolveMainServiceEnabled(conf, mainService)
	values["main_service_port"] = resolveMainServicePort(conf)

	values["os/arch"] = "x86-64"
	if runtime.GOARCH == "arm64" {
		values["os/arch"] = "aarch64"
	}

	if usr, err := user.Current(); err == nil {
		values["os/user/uid"] = usr.Uid
		values["os/user/name"] = usr.Username
		values["os/user/guid"] = usr.Gid
		values["os/user/ugroup"] = usr.Username
		if group, err := user.LookupGroupId(usr.Gid); err == nil && group != nil {
			values["os/user/ugroup"] = group.Name
		}
	} else {
		logger.Fatal(err)
	}

	for key, value := range extra {
		values[key] = value
	}

	return values
}

// renderData carries the one thing a flat configuration cannot: an ordered list.
//
// The hosts arrive in the config as nginx/hosts/<code>/name, which the tree
// would give a template as a map — fine to range over, useless for "the first
// host", which the nginx and grafana templates both ask. A slice answers both.
//
// It is never empty. A project with no host configured had a default invented
// for it in three separate places, each with its own spelling of the same
// string; inventing it once here means a template can index the list without
// asking whether there is anything in it.
func renderData(projectName string, conf map[string]string) map[string]any {
	hosts := configs.GetHosts(conf)
	if len(hosts) == 0 {
		hosts = []map[string]string{{"name": "loc." + strings.ToLower(projectName) + ".com", "code": "base"}}
	}

	return map[string]any{"nginx/hosts": hosts}
}
