// Package proxytransform lets an installation post-process the fully assembled
// nginx proxy.conf before it is written to disk.
//
// Symmetric with src/helper/dockertransform — same pattern, narrower scope.
package proxytransform

// ProxyConfTransformer rewrites the proxy.conf content right before it lands
// on disk. Receives the full file (all per-project server blocks + the
// default-server fallback already concatenated). Must return a valid nginx
// config; an empty return is treated as "no change".
type ProxyConfTransformer interface {
	TransformProxyConf(content string) string
}

// ProjectProxyConfTransformer rewrites one project's server blocks and is told
// which project they belong to.
//
// The interface above hands over the finished file and says nothing about whose
// blocks are in it, so a transformer that needs to know had one place left to
// ask: the working directory. That is not the same question. `proxy.conf` holds
// every project at once and is regenerated on behalf of a project the caller
// may not be standing in — during a removal it is regenerated on behalf of
// whichever project is left — so the answer was whatever directory the command
// ran in.
//
// Measured on a production installation on 2026-08-31: forty service locations
// across seven projects, all carrying one suffix, because one project's
// configuration had been applied to the whole file. The same run minted and
// persisted configuration for a directory that was not a project at all — a
// home directory the command happened to be started from — turning it into a
// registry entry.
type ProjectProxyConfTransformer interface {
	TransformProjectProxyConf(projectName, content string) string
}

// transformers run in registration order, each seeing the previous one's
// output.
//
// This used to be a single slot with last-writer-wins semantics, which is fine
// for one consumer and a trap for two: the second registration silently
// disabled the first, and the symptom would have been a proxy.conf missing
// whichever rewrite lost the race. Two independent consumers are the normal
// case — routing and TLS have nothing to do with each other and both need the
// generated file.
var transformers []ProxyConfTransformer

// AddProxyConfTransformer appends a transformer to the chain. Order is
// registration order, so a transformer that depends on another's output
// registers after it.
func AddProxyConfTransformer(t ProxyConfTransformer) {
	if t != nil {
		transformers = append(transformers, t)
	}
}

var projectTransformers []ProjectProxyConfTransformer

// AddProjectProxyConfTransformer appends a per-project transformer. Same
// ordering rule as the whole-file chain, and the two are independent: a
// per-project transformer sees one project's blocks before they are
// concatenated, a whole-file one sees the assembled result.
func AddProjectProxyConfTransformer(t ProjectProxyConfTransformer) {
	if t != nil {
		projectTransformers = append(projectTransformers, t)
	}
}

// ApplyProject runs the per-project transformers over one project's blocks.
//
// An empty project name runs nothing: the whole point of this path is that the
// caller knows whose blocks these are, and a transformer asked to rewrite
// "somebody's" configuration is the defect being fixed, not a fallback.
func ApplyProject(projectName, content string) string {
	if projectName == "" {
		return content
	}

	for _, t := range projectTransformers {
		out := t.TransformProjectProxyConf(projectName, content)
		if out != "" {
			content = out
		}
	}

	return content
}

// SetProxyConfTransformer replaces the whole chain with one transformer.
// Kept for callers that mean "this and nothing else"; anything wanting to
// coexist should use AddProxyConfTransformer.
func SetProxyConfTransformer(t ProxyConfTransformer) {
	if t == nil {
		transformers = nil
		return
	}
	transformers = []ProxyConfTransformer{t}
}

// ResetTransformers clears both chains. For tests, which would otherwise leak
// registrations into each other.
func ResetTransformers() {
	transformers = nil
	projectTransformers = nil
}

// Apply runs the registered transformers in order. A transformer returning an
// empty string is treated as "no change" and does not truncate the file for
// the ones after it.
func Apply(content string) string {
	for _, t := range transformers {
		out := t.TransformProxyConf(content)
		if out != "" {
			content = out
		}
	}
	return content
}
