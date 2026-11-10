// Package buildcache points the image builds of a generated compose file at a
// shared BuildKit cache, for CI.
//
// Every end-to-end shard runs on a fresh runner and built every image from
// nothing: measured on 2026-11-10, a project's first setup took 113 s for
// `language=none` and 276 s for `language=php` against 11 to 14 s with a warm
// build cache, and those first builds were about 39 of the 95 minutes of tests
// in a run. The downloads were already cached (squid, 99.5 % hits); what is left
// is running the build steps themselves.
//
// With MADOCK_BUILD_CACHE=gha each `build:` block gets a cache_from and a
// cache_to on GitHub's cache, scoped by the Dockerfile and the build block it
// is built with, so the same image built by another test or another shard is
// taken from the cache. Off — the variable unset, as everywhere but CI — the
// compose file is not touched.
package buildcache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// Env names the variable that turns it on. Its only value is "gha".
const Env = "MADOCK_BUILD_CACHE"

// Enabled reports whether this process should add the cache.
func Enabled() bool {
	return os.Getenv(Env) == "gha"
}

// Apply returns compose with a cache on every service built from a Dockerfile.
// read gives the Dockerfile's content for a build context and file name, as
// written in the compose file; "" when it cannot be read, which leaves that
// service without a cache rather than with a scope that matches anything.
func Apply(compose string, read func(context, dockerfile string) string) string {
	lines := strings.Split(compose, "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		out = append(out, lines[i])
		if lines[i] != "    build:" {
			continue
		}
		// The block: every following line indented deeper than `build:`.
		j := i + 1
		var context, dockerfile string
		var block []string
		hasCache := false
		for ; j < len(lines) && indent(lines[j]) > 4; j++ {
			trimmed := strings.TrimSpace(lines[j])
			switch {
			case strings.HasPrefix(trimmed, "context:"):
				context = strings.TrimSpace(strings.TrimPrefix(trimmed, "context:"))
			case strings.HasPrefix(trimmed, "dockerfile:"):
				dockerfile = strings.TrimSpace(strings.TrimPrefix(trimmed, "dockerfile:"))
			case strings.HasPrefix(trimmed, "cache_from:"), strings.HasPrefix(trimmed, "cache_to:"):
				hasCache = true
			}
			block = append(block, lines[j])
		}
		out = append(out, block...)
		i = j - 1
		if hasCache || context == "" || dockerfile == "" {
			continue
		}
		content := read(context, dockerfile)
		if content == "" {
			continue
		}
		scope := Scope(dockerfile, content, strings.Join(block, "\n"))
		out = append(out,
			"      cache_from:",
			"        - type=gha,scope="+scope,
			"      cache_to:",
			"        - type=gha,scope="+scope+",mode=min",
		)
	}
	return strings.Join(out, "\n")
}

// Scope names a cache by what decides the image: the Dockerfile's content and
// the build block, so two projects building the same thing share it and a
// different Dockerfile never reads another's layers.
func Scope(dockerfile, content, block string) string {
	sum := sha256.Sum256([]byte(content + "\x00" + block))
	name := strings.TrimSuffix(filepath.Base(dockerfile), ".Dockerfile")
	return "madock-" + name + "-" + hex.EncodeToString(sum[:])[:16]
}

func indent(line string) int {
	if strings.TrimSpace(line) == "" {
		return 99 // a blank line inside a block does not end it
	}
	return len(line) - len(strings.TrimLeft(line, " "))
}
