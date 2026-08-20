package versions

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/paths"
)

// renames is every spelling that moved in 3.9.8, in both the form a config
// file uses and the form a template uses.
//
// Node stopped being PHP's business. `php/nodejs/enabled` said that a runtime
// belongs to the language beside it, and it does not: a Python service with a
// JavaScript front end, or a Go one with an admin panel, needs exactly the same
// thing. Half the design was already general — the version has always lived at
// `nodejs/version`, and the php snippet read it — so only the switch was wrong.
//
// `php/yarn/enabled` goes with it, and that one was never even declared: no
// default carried it, three platform configurators set it, and one template
// read it, while a real `nodejs/yarn/enabled` sat in the defaults unused.
var renames = [][2]string{
	{"php/nodejs/enabled", "nodejs/embedded/enabled"},
	{"php/yarn/enabled", "nodejs/yarn/enabled"},
	// The template spelling of the same two, for copies of madock's own
	// templates that a project keeps under .madock/docker/.
	{".php.nodejs.enabled", ".nodejs.embedded.enabled"},
	{".php.yarn.enabled", ".nodejs.yarn.enabled"},
}

// V398 carries the rename into every place a copy of the old name can be.
//
// There are more of those than there look to be, and missing one is silent in
// the worst way: node quietly stops being installed, and the failure surfaces
// later as a build that cannot find npm.
//
//  1. the installation's own config.xml, and the global project defaults
//     beside it — a machine-wide answer lives in either
//  2. each project's registry config
//  3. each project's own .madock/config.xml. **This file is otherwise never
//     written by madock** — it is committed to the project's repository and no
//     command may touch it. A migration is the one exception, because a
//     renamed key has to move and leaving it behind would drop the setting
//     without a word
//  4. template overrides under {project}/.madock/docker/, which are copies of
//     madock's own templates and therefore still say `.php.nodejs.enabled`
//
// What is deliberately not here: the extracted templates under
// {execDir}/docker/. Those are rewritten from the binary on the next run, so
// they heal themselves; editing them would be undone anyway.
func V398() {
	execDir := paths.GetExecDirPath()

	// 1. The installation, and the defaults every project starts from.
	for _, file := range []string{
		execDir + "/config.xml",
		execDir + "/projects/config.xml",
	} {
		renameInConfig(file)
	}

	projectsDir := execDir + "/projects"
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectName := entry.Name()
		configPath := projectsDir + "/" + projectName + "/config.xml"
		if !paths.IsFileExist(configPath) {
			continue
		}

		// 2. The registry copy.
		renameInConfig(configPath)

		projectPath := configs.GetProjectConfigOnly(projectName)["path"]
		if projectPath == "" {
			continue
		}

		// 3. The project's own, which nothing else here is allowed to write.
		renameInConfig(projectPath + "/.madock/config.xml")

		// 4. And any template a project copied out of madock to override.
		renameInTemplates(projectPath + "/.madock/docker")
	}
}

// renameInConfig moves the renamed keys inside one config file.
//
// It parses and writes the file back rather than editing its text, because
// nesting makes textual surgery unsafe: <enabled> under <php><nodejs> and
// <enabled> under <nodejs> are the same eight characters, and a regexp that
// tells them apart is a regexp nobody can check.
//
// What that costs: XML comments in the file do not survive. These files are
// written by madock itself and carry none — checked across the projects on this
// machine — but a hand-written one would lose them, which is worth knowing
// before this pattern is copied. Everything else survives, including a
// top-level key outside <scopes> such as allow_destructive_commands, because
// the parser keeps it and the writer puts it back.
//
// The file is only rewritten when something actually moved, so a migration that
// has nothing to do leaves no diff.
func renameInConfig(path string) {
	if !paths.IsFileExist(path) {
		return
	}

	parsed := configs.ParseXmlFile(path)
	moved := false

	for _, pair := range renames {
		if strings.HasPrefix(pair[0], ".") {
			continue // template spelling; not what a config file uses
		}
		if renameKey(parsed, pair[0], pair[1]) {
			moved = true
		}
	}

	if !moved {
		return
	}

	generic := make(map[string]interface{}, len(parsed))
	for key, value := range parsed {
		generic[key] = value
	}

	_ = os.WriteFile(path, []byte(configs.RenderXml(generic)), 0644)
}

// renameKey moves one key within every scope the file holds, and reports
// whether it moved anything.
//
// Scopes are walked rather than assumed: a project can carry several, and a
// rename that only fixed `default` would leave the others reading a key that no
// longer exists — which the renderer answers as false, silently.
func renameKey(parsed map[string]string, oldKey, newKey string) bool {
	moved := false

	for key, value := range parsed {
		scope, rest, ok := scopeOf(key)
		if !ok || rest != oldKey {
			continue
		}
		if _, taken := parsed[scope+newKey]; taken {
			// Somebody already set the new name. Theirs is the deliberate one;
			// the old key is dropped rather than allowed to win.
			delete(parsed, key)
			moved = true
			continue
		}
		parsed[scope+newKey] = value
		delete(parsed, key)
		moved = true
	}

	return moved
}

// scopeOf splits "scopes/<name>/rest" into its prefix and the rest.
func scopeOf(key string) (string, string, bool) {
	if !strings.HasPrefix(key, "scopes/") {
		return "", "", false
	}
	rest := strings.TrimPrefix(key, "scopes/")
	name, tail, ok := strings.Cut(rest, "/")
	if !ok {
		return "", "", false
	}
	return "scopes/" + name + "/", tail, true
}

// renameInTemplates walks a project's own copies of madock's templates.
//
// These are the ones nothing else would ever fix. A project overrides a
// template by copying it into .madock/docker/ and editing it, and that copy
// still asks about `.php.nodejs.enabled` — a key that no longer exists, which
// the renderer answers as false. Node then stops being installed and nothing
// says so.
func renameInTemplates(root string) {
	if !paths.IsFileExist(root) {
		return
	}

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		updated := string(body)
		for _, pair := range renames {
			if !strings.HasPrefix(pair[0], ".") {
				continue // config spellings do not appear in templates
			}
			updated = strings.ReplaceAll(updated, pair[0], pair[1])
		}

		if updated != string(body) {
			_ = os.WriteFile(path, []byte(updated), 0644)
		}
		return nil
	})
}
