package project

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/faradey/madock/v4/src/helper/paths"
)

// The whole-stack fingerprint answers "did anything change". It cannot answer
// "what", and the cost of that was measured on a production store on
// 2026-09-21: a deploy whose only change was the nginx vhost recreated every
// container — MariaDB and OpenSearch restarted, five `SQLSTATE[HY000] [2002]
// Connection refused` in the application log, a 500 to a live visitor — about
// sixty seconds without a database for a file nginx can reload in place.
//
// This file records the stack per service instead, and in two halves, because
// they call for different actions:
//
//   - definition — the service's block in the resolved compose config, its
//     Dockerfile, and whatever that Dockerfile copies in. A change here needs
//     the container recreated (and the image rebuilt).
//   - mounts — the generated files bind-mounted into it from the build
//     context. Those are already inside the running container; the process
//     has to re-read them. For nginx that is a reload; for anything else a
//     restart.
//
// Everything outside `services` — networks, named volumes, the project name —
// is one global hash, and a change there recreates the stack as before.
//
// The resolved config comes from `docker compose config`, not from our own
// templates: it is the merge of the compose file, the OS override and anything
// an edition's transformer added, which is the only shape docker itself acts
// on.
type serviceRecord struct {
	Definition string `json:"definition"`
	Mounts     string `json:"mounts"`
}

type stackRecord struct {
	Global   string                   `json:"global"`
	Services map[string]serviceRecord `json:"services"`
}

// StackDiff is what changed since the containers were created, sorted by the
// action it calls for.
type StackDiff struct {
	// Known is false when the question could not be answered — no record from
	// last time, no compose config, an unreadable file. The caller then does
	// what it always did: recreate everything.
	Known bool
	// Global is a change outside any one service. Recreate everything.
	Global bool
	// Recreate names services whose definition changed.
	Recreate []string
	// Restart names services whose mounted files changed and nothing else.
	// nginx is listed here too; the caller decides that nginx reloads.
	Restart []string
	// Removed names services the record knew and the stack no longer declares.
	Removed []string
}

// Narrow reports whether the diff names specific services rather than the
// whole stack — the case where recreating everything would be waste.
func (d StackDiff) Narrow() bool {
	return d.Known && !d.Global && len(d.Removed) == 0 &&
		(len(d.Recreate) > 0 || len(d.Restart) > 0)
}

// Empty reports that the record and the stack agree on every service.
func (d StackDiff) Empty() bool {
	return d.Known && !d.Global && len(d.Recreate) == 0 && len(d.Restart) == 0 && len(d.Removed) == 0
}

// composeConfigJSON resolves the project's compose files the way docker does.
// A variable so a test can hand in a config without a docker daemon.
var composeConfigJSON = func(projectName string) ([]byte, error) {
	pp := paths.NewProjectPaths(projectName)
	return exec.Command("docker", "compose",
		"-f", pp.DockerCompose(), "-f", pp.DockerComposeOverride(),
		"config", "--format", "json").Output()
}

// DiffStack compares the stack as rendered now with the record of the stack
// the containers were created from.
func DiffStack(projectName string) StackDiff {
	current, err := currentStackRecord(projectName)
	if err != nil {
		return StackDiff{}
	}
	previous, err := readStackRecord(projectName)
	if err != nil {
		return StackDiff{}
	}
	return diffStacks(previous, current)
}

// diffStacks is the comparison on its own, so it can be tested on two records
// without a project.
func diffStacks(previous, current stackRecord) StackDiff {
	d := StackDiff{Known: true}
	if previous.Global != current.Global {
		d.Global = true
	}
	for name, now := range current.Services {
		was, known := previous.Services[name]
		switch {
		case !known:
			// A service the record has never seen: create it. `up` on a name
			// with no container creates it, so it goes with the recreates.
			d.Recreate = append(d.Recreate, name)
		case was.Definition != now.Definition:
			d.Recreate = append(d.Recreate, name)
		case was.Mounts != now.Mounts:
			d.Restart = append(d.Restart, name)
		}
	}
	for name := range previous.Services {
		if _, still := current.Services[name]; !still {
			d.Removed = append(d.Removed, name)
		}
	}
	sort.Strings(d.Recreate)
	sort.Strings(d.Restart)
	sort.Strings(d.Removed)
	return d
}

// RecordStack stores the per-service record of the stack that was just
// applied. Called next to RecordApplied; the two records answer different
// questions and one is not derived from the other.
func RecordStack(projectName string) {
	record, err := currentStackRecord(projectName)
	if err != nil {
		return
	}
	data, err := json.Marshal(record)
	if err != nil {
		return
	}
	_ = os.WriteFile(stackRecordPath(projectName), data, 0o644)
}

func stackRecordPath(projectName string) string {
	return filepath.Join(paths.MakeDirsByPath(paths.CacheDir()), strings.ToLower(projectName)+"-stack-services.json")
}

func readStackRecord(projectName string) (stackRecord, error) {
	var record stackRecord
	data, err := os.ReadFile(stackRecordPath(projectName))
	if err != nil {
		return record, err
	}
	if err := json.Unmarshal(data, &record); err != nil {
		return record, err
	}
	return record, nil
}

func currentStackRecord(projectName string) (stackRecord, error) {
	data, err := composeConfigJSON(projectName)
	if err != nil {
		return stackRecord{}, err
	}
	pp := paths.NewProjectPaths(projectName)
	return recordFromComposeConfig(data, pp.RuntimeDir())
}

// composeService is the part of a resolved service block this file reads.
// Everything else in the block still counts — through the raw bytes — it just
// does not need a name.
type composeService struct {
	Build   json.RawMessage `json:"build"`
	Volumes []struct {
		Type   string `json:"type"`
		Source string `json:"source"`
	} `json:"volumes"`
}

// recordFromComposeConfig builds the record from a resolved compose config and
// the runtime directory whose files that config mounts and builds from.
func recordFromComposeConfig(configJSON []byte, runtimeDir string) (stackRecord, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(configJSON, &top); err != nil {
		return stackRecord{}, err
	}

	var services map[string]json.RawMessage
	if raw, ok := top["services"]; ok {
		if err := json.Unmarshal(raw, &services); err != nil {
			return stackRecord{}, err
		}
		delete(top, "services")
	}
	// json.Marshal writes map keys sorted, which is what makes this stable.
	global, err := json.Marshal(top)
	if err != nil {
		return stackRecord{}, err
	}

	record := stackRecord{
		Global:   sum(global),
		Services: make(map[string]serviceRecord, len(services)),
	}
	ctxDir := filepath.Clean(filepath.Join(runtimeDir, "ctx"))

	for name, raw := range services {
		var svc composeService
		if err := json.Unmarshal(raw, &svc); err != nil {
			return stackRecord{}, err
		}

		definition := sha256.New()
		definition.Write(raw)
		if err := hashBuildInputs(definition, svc.Build); err != nil {
			return stackRecord{}, err
		}

		mounts := sha256.New()
		for _, v := range svc.Volumes {
			if v.Type != "bind" {
				continue
			}
			source := v.Source
			// `docker compose config` writes bind sources absolute; a relative
			// one is relative to the compose file, which lives in the runtime
			// directory. Left relative it would match nothing below and a
			// mounted file would silently drop out of the record.
			if !filepath.IsAbs(source) {
				source = filepath.Join(runtimeDir, source)
			}
			source = filepath.Clean(source)
			// Only what MakeConf generated. The source tree, the composer home
			// and ~/.ssh are mounted too, and hashing them would make every
			// edit of the application a reason to restart a container.
			if source != ctxDir && !strings.HasPrefix(source, ctxDir+string(filepath.Separator)) {
				continue
			}
			if err := hashPath(mounts, source); err != nil {
				return stackRecord{}, err
			}
		}

		record.Services[name] = serviceRecord{
			Definition: hex.EncodeToString(definition.Sum(nil)),
			Mounts:     hex.EncodeToString(mounts.Sum(nil)),
		}
	}
	return record, nil
}

// hashBuildInputs adds the Dockerfile and whatever it copies into the image.
//
// The resolved config writes `build` as an object with `context` and
// `dockerfile`; the short string form is a context alone. Only the files the
// Dockerfile names are hashed, not the whole context: every service of a
// project builds from the same `ctx`, so hashing it whole would make one
// service's change everyone's.
func hashBuildInputs(h interface{ Write([]byte) (int, error) }, build json.RawMessage) error {
	if len(build) == 0 || string(build) == "null" {
		return nil
	}
	context, dockerfile := "", "Dockerfile"
	var asString string
	if err := json.Unmarshal(build, &asString); err == nil {
		context = asString
	} else {
		var asObject struct {
			Context    string `json:"context"`
			Dockerfile string `json:"dockerfile"`
		}
		if err := json.Unmarshal(build, &asObject); err != nil {
			return err
		}
		context = asObject.Context
		if asObject.Dockerfile != "" {
			dockerfile = asObject.Dockerfile
		}
	}
	if context == "" {
		return nil
	}
	if !filepath.IsAbs(dockerfile) {
		dockerfile = filepath.Join(context, dockerfile)
	}
	content, err := os.ReadFile(dockerfile)
	if err != nil {
		// A Dockerfile that is not there is a build that will fail on its own
		// terms; not this check's finding.
		return nil
	}
	h.Write([]byte(dockerfile + "\x00"))
	h.Write(content)
	h.Write([]byte{0})

	for _, source := range copiedSources(string(content)) {
		// A COPY source may be a pattern — `COPY scripts/*.sh /usr/local/bin/`
		// — and a pattern hashed as a literal path matches nothing, so the
		// files behind it would change without the record noticing.
		matches, _ := filepath.Glob(filepath.Join(context, source))
		if matches == nil {
			matches = []string{filepath.Join(context, source)}
		}
		sort.Strings(matches)
		for _, match := range matches {
			if err := hashPath(h, match); err != nil {
				// Missing sources fail the build, not the diff.
				continue
			}
		}
	}
	return nil
}

// copiedSources lists the context-relative sources of every COPY and ADD in a
// Dockerfile. `--from=` stages copy from another image, not the context, and a
// URL is not a file; both are left out. The last operand is the destination.
func copiedSources(dockerfile string) []string {
	var sources []string
	for _, line := range strings.Split(dockerfile, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		verb := strings.ToUpper(fields[0])
		if verb != "COPY" && verb != "ADD" {
			continue
		}
		operands := fields[1 : len(fields)-1]
		fromStage := false
		for _, op := range operands {
			if strings.HasPrefix(op, "--from=") {
				fromStage = true
			}
		}
		if fromStage {
			continue
		}
		for _, op := range operands {
			if strings.HasPrefix(op, "--") || strings.Contains(op, "://") {
				continue
			}
			sources = append(sources, op)
		}
	}
	return sources
}

// hashPath adds a file, or every file under a directory, to h.
func hashPath(h interface{ Write([]byte) (int, error) }, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(path + "\x00"))
		h.Write(content)
		h.Write([]byte{0})
		return nil
	}
	var files []string
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		files = append(files, p)
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(files)
	// The directory's name is hashed even when it is empty, so a directory
	// that gained or lost its last file is a change.
	h.Write([]byte(path + "/\x00"))
	for _, f := range files {
		if err := hashPath(h, f); err != nil {
			return err
		}
	}
	return nil
}

func sum(data []byte) string {
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:])
}
