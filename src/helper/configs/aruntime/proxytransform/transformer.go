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
