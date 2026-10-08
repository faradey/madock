package project

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/tmpl"
)

// warnUnknownKeys says out loud when a project's own template reads a setting
// this madock does not have.
//
// An absent key renders as empty — it has to, because madock's shared snippets
// ask about services a platform has never heard of — so a template written for
// a newer madock, or one carrying a name madock dropped long ago, renders
// without a word and produces something wrong. Measured on this machine on
// 2026-10-08: three of eight projects with their own templates read
// nginx/port/project and nginx/port/project_ssl, the port arithmetic madock
// stopped supporting years ago, and their compose files publish ":80" and
// ":443" with no host port at all.
//
// A warning and not a refusal, deliberately. Those three projects start today
// and a refusal would stop them; the same is true of a project whose templates
// ask for a madock-pro setting while someone runs it with madock. The check is
// only over the project's own copies — what madock ships has its own test —
// and each file is reported once per run.
func warnUnknownKeys(projectName, path, body string, values map[string]string, data map[string]any) {
	if !isProjectOwnedTemplate(projectName, path) {
		return
	}
	if _, done := warnedUnknownKeys.LoadOrStore(path, true); done {
		return
	}

	names := unknownKeysIn(path, body, values, data)
	if len(names) == 0 {
		return
	}

	fmtc.WarningLn(path + " reads settings this madock does not have, so they render empty:")
	for _, key := range names {
		fmtc.WarningLn("  " + key)
	}
	fmtc.ToDoLn("A misspelt or retired name, a setting of a newer madock, or one of madock-pro. Check the name against `madock config:list`.")
}

var warnedUnknownKeys sync.Map

// unknownKeysIn lists, sorted, the keys a template reads that the render does
// not answer. Old-syntax templates are converted first, the way the render does.
func unknownKeysIn(path, body string, values map[string]string, data map[string]any) []string {
	source, _ := tmpl.Legacy(body)
	keys, err := tmpl.Keys(path, source)
	if err != nil {
		return nil // the render itself reports a template that does not parse
	}

	unknown := map[string]bool{}
	for _, key := range keys {
		if !knownTo(key, values, data) {
			unknown[key] = true
		}
	}

	names := make([]string, 0, len(unknown))
	for key := range unknown {
		names = append(names, key)
	}
	sort.Strings(names)
	return names
}

// knownTo reports whether a template key is something the render answers: a
// setting, a computed value, an ordered list, or the parent of settings whose
// children are named by whoever writes them (worker/programs/<name>/...).
func knownTo(key string, values map[string]string, data map[string]any) bool {
	if _, ok := values[key]; ok {
		return true
	}
	if _, ok := data[key]; ok {
		return true
	}
	prefix := key + "/"
	for name := range values {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func isProjectOwnedTemplate(projectName, path string) bool {
	clean := filepath.Clean(path)
	for _, dir := range projectOwnedTemplateDirs(projectName) {
		if strings.HasPrefix(clean, filepath.Clean(dir)+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
