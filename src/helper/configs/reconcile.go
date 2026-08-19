package configs

import (
	"sort"
	"strings"

	"github.com/faradey/madock/v3/src/helper/paths"
)

// ReconcileResult is what one reconciliation did, and it is meant to be printed.
//
// Silence here would recreate the defect: a setting disappearing from a machine
// without anybody being told is the same class of failure as one that refuses to.
type ReconcileResult struct {
	// Baseline is set on the first run, where there is nothing to compare
	// against yet. Nothing is removed and the current project config is recorded
	// as the starting point.
	Baseline bool
	// Removed are the keys the project deleted and this machine still carried.
	Removed []string
	// Kept are the keys the project deleted while this machine holds a different
	// value — set here, by a person, after the copy was made. Reported and left
	// alone: dropping somebody's local change is not what "the project removed a
	// default" asks for.
	Kept []string
}

// snapshotFileName is the copy of the project's config as madock last saw it.
//
// It lives in the installation, never in the project, and is not a second
// configuration: nothing reads a value out of it. It answers one question, the
// one nothing could answer before — of the keys in the installed copy, which
// came from the project?
const snapshotFileName = "project-config-snapshot.xml"

// ReconcileRemovedProjectKeys drops from the installed configuration the keys
// that have since been deleted from the project's own.
//
// madock keeps two copies of a project's configuration: the one in the
// repository, `<project>/.madock/config.xml`, and the one in the installation,
// written at setup. Reads prefer the first, so an added or changed setting
// arrives on its own — and a *deleted* one does not, because the read falls
// through to the installed copy, which still holds it. Measured on Pricesmith on
// 2026-08-17: a `custom_commands` block was removed from the repository,
// committed and rolled out, and `madock pr` went on working on every machine
// that had ever run setup. Nothing failed and nobody was told.
//
// The obstacle was never the deletion, it was telling two identical-looking keys
// apart: one copied from the project at setup, one typed here with `config:set`.
// A snapshot of the project's config as madock last saw it answers exactly that,
// the way a merge base does — a key that was in the snapshot and is gone from the
// project was the project's to remove; a key whose installed value no longer
// matches the snapshot was changed on this machine and is not.
//
// Conservative by construction: only keys the snapshot recorded are ever
// candidates, so anything madock writes into the installed copy itself — `path`,
// the generated passwords, whatever a person set — is never a candidate at all.
func ReconcileRemovedProjectKeys(projectName string) (ReconcileResult, error) {
	if projectName == "" {
		projectName = GetProjectName()
	}

	projectDir := paths.GetExecDirPath() + "/projects/" + projectName
	runtimeFile := projectDir + "/config.xml"
	if !paths.IsFileExist(runtimeFile) {
		return ReconcileResult{}, nil
	}

	projectPath := strings.TrimSpace(GetProjectConfigOnly(projectName)["path"])
	if projectPath == "" {
		return ReconcileResult{}, nil
	}
	projectFile := projectPath + "/.madock/config.xml"
	if !paths.IsFileExist(projectFile) {
		// A project that keeps no config of its own has nothing to remove from
		// here, and recording a snapshot for it would only invite one later.
		return ReconcileResult{}, nil
	}

	snapshotFile := projectDir + "/" + snapshotFileName
	if !paths.IsFileExist(snapshotFile) {
		if err := paths.Copy(projectFile, snapshotFile); err != nil {
			return ReconcileResult{}, err
		}
		return ReconcileResult{Baseline: true}, nil
	}

	result := compareForRemoval(
		ParseXmlFile(snapshotFile),
		ParseXmlFile(projectFile),
		ParseXmlFile(runtimeFile),
	)

	if len(result.Removed) > 0 {
		if err := removeScopedKeys(runtimeFile, result.Removed); err != nil {
			return result, err
		}
	}

	if err := paths.Copy(projectFile, snapshotFile); err != nil {
		return result, err
	}
	CleanCache()

	return result, nil
}

// compareForRemoval is the whole decision, kept apart from the files so it can
// be read and tested as the rule it is.
func compareForRemoval(snapshot, current, runtime map[string]string) ReconcileResult {
	var result ReconcileResult

	for key, wasValue := range snapshot {
		// Only settings. `activeScope` and anything else outside a scope is
		// structure, not a value a project removes.
		if !strings.HasPrefix(key, "scopes/") {
			continue
		}
		if _, stillThere := current[key]; stillThere {
			continue
		}
		nowValue, inRuntime := runtime[key]
		if !inRuntime {
			// Already gone from the installed copy — nothing to do, and saying
			// so would be noise.
			continue
		}
		if nowValue != wasValue {
			result.Kept = append(result.Kept, key)
			continue
		}
		result.Removed = append(result.Removed, key)
	}

	sort.Strings(result.Removed)
	sort.Strings(result.Kept)
	return result
}

// removeScopedKeys deletes the keys from the installed config, one scope at a
// time, because the editor takes keys relative to a scope.
func removeScopedKeys(file string, keys []string) error {
	byScope := map[string][]string{}
	for _, key := range keys {
		scope, rest, ok := splitScopedKey(key)
		if !ok {
			continue
		}
		byScope[scope] = append(byScope[scope], rest)
	}

	scopes := make([]string, 0, len(byScope))
	for scope := range byScope {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)

	for _, scope := range scopes {
		if err := RemoveKeepingComments(file, byScope[scope], scope); err != nil {
			return err
		}
	}
	return nil
}

// splitScopedKey turns `scopes/default/db/enabled` into `default` and
// `db/enabled`.
func splitScopedKey(key string) (scope, rest string, ok bool) {
	trimmed := strings.TrimPrefix(key, "scopes/")
	if trimmed == key {
		return "", "", false
	}
	slash := strings.IndexByte(trimmed, '/')
	if slash <= 0 || slash == len(trimmed)-1 {
		return "", "", false
	}
	return trimmed[:slash], trimmed[slash+1:], true
}
