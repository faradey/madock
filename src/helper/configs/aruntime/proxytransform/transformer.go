// Package proxytransform exposes a single hook that lets enterprise (or any
// downstream consumer) post-process the fully assembled nginx proxy.conf
// before it is written to disk.
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

var transformer ProxyConfTransformer

// SetProxyConfTransformer registers a custom transformer. Last writer wins
// (mirrors SetDockerTransformer semantics).
func SetProxyConfTransformer(t ProxyConfTransformer) {
	transformer = t
}

// Apply runs the registered transformer if any. Returns content unchanged
// when no transformer is set or when the transformer returns an empty string.
func Apply(content string) string {
	if transformer == nil {
		return content
	}
	out := transformer.TransformProxyConf(content)
	if out == "" {
		return content
	}
	return out
}
